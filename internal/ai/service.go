package ai

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"docko/internal/database"
	"docko/internal/database/sqlc"
)

// Service orchestrates AI providers and manages suggestions
type Service struct {
	db        *database.DB
	providers []Provider // In priority order
}

// NewService creates a new AI service with all available providers
func NewService(db *database.DB) *Service {
	providers := []Provider{
		NewOpenAIProvider(),
		NewAnthropicProvider(),
		NewOllamaProvider(),
	}

	// Log available providers
	for _, p := range providers {
		if p.Available() {
			slog.Info("AI provider available", "provider", p.Name())
		}
	}

	return &Service{
		db:        db,
		providers: providers,
	}
}

// GetSettings returns current AI settings
func (s *Service) GetSettings(ctx context.Context) (sqlc.AiSetting, error) {
	return s.db.Queries.GetAISettings(ctx)
}

// UpdateSettings updates AI settings
func (s *Service) UpdateSettings(ctx context.Context, params sqlc.UpdateAISettingsParams) (sqlc.AiSetting, error) {
	return s.db.Queries.UpdateAISettings(ctx, params)
}

// AnalyzeResult contains the results of document analysis
type AnalyzeResult struct {
	AutoApplied int // Count of auto-applied suggestions
	Pending     int // Count of pending suggestions
	Skipped     int // Count of low-confidence suggestions skipped
	Provider    string
	Duration    time.Duration
}

// AnalyzeDocument analyzes a document using AI and stores suggestions
func (s *Service) AnalyzeDocument(ctx context.Context, docID uuid.UUID, jobID *uuid.UUID) (*AnalyzeResult, error) {
	start := time.Now()

	// Get AI settings for thresholds
	settings, err := s.db.Queries.GetAISettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("get ai settings: %w", err)
	}

	// Get document with text content
	doc, err := s.db.Queries.GetDocument(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}

	if doc.TextContent == nil || *doc.TextContent == "" {
		return nil, fmt.Errorf("document has no text content")
	}

	// Get existing tags and correspondents for context
	existingTags, err := s.getExistingTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("get existing tags: %w", err)
	}

	existingCorrespondents, err := s.getExistingCorrespondents(ctx)
	if err != nil {
		return nil, fmt.Errorf("get existing correspondents: %w", err)
	}

	// Truncate text to first N pages worth (roughly 3000 chars per page)
	maxPages := int(settings.MaxPages)
	maxChars := maxPages * 3000
	textContent := *doc.TextContent
	if len(textContent) > maxChars {
		textContent = textContent[:maxChars]
	}

	// Build analysis request
	req := AnalyzeRequest{
		DocumentID:     docID,
		TextContent:    textContent,
		ExistingTags:   existingTags,
		Correspondents: existingCorrespondents,
	}

	// Get preferred provider or use fallback
	provider, err := s.selectProvider(settings.PreferredProvider)
	if err != nil {
		return nil, fmt.Errorf("no available AI provider: %w", err)
	}

	slog.Info("analyzing document with AI",
		"doc_id", docID,
		"provider", provider.Name(),
		"text_length", len(textContent))

	// Call AI provider
	resp, err := provider.Analyze(ctx, req)
	if err != nil {
		// Try fallback providers
		resp, provider, err = s.tryFallbackProviders(ctx, req, provider.Name())
		if err != nil {
			return nil, fmt.Errorf("all providers failed: %w", err)
		}
	}

	// Track usage
	jobUUID := pgtype.UUID{}
	if jobID != nil {
		jobUUID.Bytes = *jobID
		jobUUID.Valid = true
	}

	_, err = s.db.Queries.CreateAIUsage(ctx, sqlc.CreateAIUsageParams{
		DocumentID:   docID,
		JobID:        jobUUID,
		Provider:     provider.Name(),
		Model:        resp.Usage.Model,
		InputTokens:  int32(resp.Usage.InputTokens),
		OutputTokens: int32(resp.Usage.OutputTokens),
	})
	if err != nil {
		slog.Warn("failed to track AI usage", "error", err)
	}

	// Convert thresholds from pgtype.Numeric to float64
	autoApplyThreshold := numericToFloat64(settings.AutoApplyThreshold)
	reviewThreshold := numericToFloat64(settings.ReviewThreshold)

	// Process suggestions
	result := &AnalyzeResult{
		Provider: provider.Name(),
		Duration: time.Since(start),
	}

	for _, suggestion := range resp.Suggestions {
		if suggestion.Confidence >= autoApplyThreshold {
			// Auto-apply high-confidence suggestions
			err := s.autoApplySuggestion(ctx, docID, jobUUID, suggestion)
			if err != nil {
				slog.Warn("failed to auto-apply suggestion",
					"doc_id", docID,
					"type", suggestion.Type,
					"value", suggestion.Value,
					"error", err)
				continue
			}
			result.AutoApplied++
		} else if suggestion.Confidence >= reviewThreshold {
			// Store as pending for review
			err := s.storePendingSuggestion(ctx, docID, jobUUID, suggestion)
			if err != nil {
				slog.Warn("failed to store pending suggestion",
					"doc_id", docID,
					"type", suggestion.Type,
					"value", suggestion.Value,
					"error", err)
				continue
			}
			result.Pending++
		} else {
			// Skip low-confidence suggestions
			result.Skipped++
			slog.Debug("skipping low-confidence suggestion",
				"doc_id", docID,
				"type", suggestion.Type,
				"value", suggestion.Value,
				"confidence", suggestion.Confidence)
		}
	}

	slog.Info("document analysis complete",
		"doc_id", docID,
		"provider", provider.Name(),
		"auto_applied", result.AutoApplied,
		"pending", result.Pending,
		"skipped", result.Skipped,
		"duration_ms", result.Duration.Milliseconds())

	return result, nil
}

// selectProvider returns the preferred provider if available, otherwise first available
func (s *Service) selectProvider(preferred *string) (Provider, error) {
	// If preferred provider specified, try it first
	if preferred != nil && *preferred != "" {
		for _, p := range s.providers {
			if p.Name() == *preferred && p.Available() {
				return p, nil
			}
		}
	}

	// Fall back to any available provider
	for _, p := range s.providers {
		if p.Available() {
			return p, nil
		}
	}

	return nil, fmt.Errorf("no AI providers available")
}

// tryFallbackProviders tries remaining providers after primary failure
func (s *Service) tryFallbackProviders(ctx context.Context, req AnalyzeRequest, failedProvider string) (*AnalyzeResponse, Provider, error) {
	var lastErr error
	for _, p := range s.providers {
		if p.Name() == failedProvider || !p.Available() {
			continue
		}

		slog.Info("trying fallback AI provider", "provider", p.Name())
		resp, err := p.Analyze(ctx, req)
		if err == nil {
			return resp, p, nil
		}
		lastErr = err
		slog.Warn("fallback provider failed", "provider", p.Name(), "error", err)
	}
	return nil, nil, lastErr
}

// autoApplySuggestion creates and applies a high-confidence suggestion
func (s *Service) autoApplySuggestion(ctx context.Context, docID uuid.UUID, jobID pgtype.UUID, suggestion Suggestion) error {
	// Start transaction
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.db.Queries.WithTx(tx)

	// Create the suggestion record
	suggestionType := sqlc.SuggestionTypeTag
	if suggestion.Type == "correspondent" {
		suggestionType = sqlc.SuggestionTypeCorrespondent
	}

	created, err := qtx.CreateAISuggestion(ctx, sqlc.CreateAISuggestionParams{
		DocumentID:     docID,
		JobID:          jobID,
		SuggestionType: suggestionType,
		Value:          suggestion.Value,
		Confidence:     float64ToNumeric(suggestion.Confidence),
		Reasoning:      &suggestion.Reasoning,
		IsNew:          suggestion.IsNew,
		Status:         sqlc.SuggestionStatusAutoApplied,
		ResolvedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		ResolvedBy:     strPtr("auto"),
	})
	if err != nil {
		return fmt.Errorf("create suggestion: %w", err)
	}

	// Apply the suggestion based on type
	if suggestion.Type == "tag" {
		err = s.applyTagSuggestion(ctx, qtx, docID, suggestion)
	} else if suggestion.Type == "correspondent" {
		err = s.applyCorrespondentSuggestion(ctx, qtx, docID, suggestion)
	}

	if err != nil {
		return fmt.Errorf("apply suggestion: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	slog.Info("auto-applied AI suggestion",
		"suggestion_id", created.ID,
		"type", suggestion.Type,
		"value", suggestion.Value,
		"confidence", suggestion.Confidence)

	return nil
}

// storePendingSuggestion stores a suggestion for manual review
func (s *Service) storePendingSuggestion(ctx context.Context, docID uuid.UUID, jobID pgtype.UUID, suggestion Suggestion) error {
	suggestionType := sqlc.SuggestionTypeTag
	if suggestion.Type == "correspondent" {
		suggestionType = sqlc.SuggestionTypeCorrespondent
	}

	_, err := s.db.Queries.CreateAISuggestion(ctx, sqlc.CreateAISuggestionParams{
		DocumentID:     docID,
		JobID:          jobID,
		SuggestionType: suggestionType,
		Value:          suggestion.Value,
		Confidence:     float64ToNumeric(suggestion.Confidence),
		Reasoning:      &suggestion.Reasoning,
		IsNew:          suggestion.IsNew,
		Status:         sqlc.SuggestionStatusPending,
		ResolvedAt:     pgtype.Timestamptz{},
		ResolvedBy:     nil,
	})
	if err != nil {
		return fmt.Errorf("create pending suggestion: %w", err)
	}

	return nil
}

// applyTagSuggestion creates or finds tag and assigns to document
func (s *Service) applyTagSuggestion(ctx context.Context, qtx *sqlc.Queries, docID uuid.UUID, suggestion Suggestion) error {
	// Try to find existing tag
	tags, err := qtx.SearchTags(ctx, suggestion.Value)
	if err != nil {
		return fmt.Errorf("search tags: %w", err)
	}

	var tagID uuid.UUID
	found := false
	for _, t := range tags {
		if t.Name == suggestion.Value {
			tagID = t.ID
			found = true
			break
		}
	}

	// Create tag if not found
	if !found {
		tag, err := qtx.CreateTag(ctx, sqlc.CreateTagParams{
			Name:  suggestion.Value,
			Color: nil,
		})
		if err != nil {
			return fmt.Errorf("create tag: %w", err)
		}
		tagID = tag.ID
		slog.Info("created new tag from AI suggestion", "tag_id", tagID, "name", suggestion.Value)
	}

	// Assign tag to document
	err = qtx.AddDocumentTag(ctx, sqlc.AddDocumentTagParams{
		DocumentID: docID,
		TagID:      tagID,
	})
	if err != nil {
		return fmt.Errorf("add document tag: %w", err)
	}

	return nil
}

// applyCorrespondentSuggestion creates or finds correspondent and assigns to document
func (s *Service) applyCorrespondentSuggestion(ctx context.Context, qtx *sqlc.Queries, docID uuid.UUID, suggestion Suggestion) error {
	// Try to find existing correspondent
	correspondents, err := qtx.SearchCorrespondents(ctx, suggestion.Value)
	if err != nil {
		return fmt.Errorf("search correspondents: %w", err)
	}

	var correspondentID uuid.UUID
	found := false
	for _, c := range correspondents {
		if c.Name == suggestion.Value {
			correspondentID = c.ID
			found = true
			break
		}
	}

	// Create correspondent if not found
	if !found {
		corr, err := qtx.CreateCorrespondent(ctx, sqlc.CreateCorrespondentParams{
			Name:  suggestion.Value,
			Notes: nil,
		})
		if err != nil {
			return fmt.Errorf("create correspondent: %w", err)
		}
		correspondentID = corr.ID
		slog.Info("created new correspondent from AI suggestion", "correspondent_id", correspondentID, "name", suggestion.Value)
	}

	// Assign correspondent to document
	err = qtx.SetDocumentCorrespondent(ctx, sqlc.SetDocumentCorrespondentParams{
		DocumentID:      docID,
		CorrespondentID: correspondentID,
	})
	if err != nil {
		return fmt.Errorf("set document correspondent: %w", err)
	}

	return nil
}

// getExistingTags returns all tag names for context
func (s *Service) getExistingTags(ctx context.Context) ([]string, error) {
	tags, err := s.db.Queries.ListTagsWithCounts(ctx)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.Name
	}
	return names, nil
}

// getExistingCorrespondents returns all correspondent names for context
func (s *Service) getExistingCorrespondents(ctx context.Context) ([]string, error) {
	correspondents, err := s.db.Queries.ListCorrespondentsWithCounts(ctx)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(correspondents))
	for i, c := range correspondents {
		names[i] = c.Name
	}
	return names, nil
}

// AnyProviderAvailable returns true if at least one AI provider is available
func (s *Service) AnyProviderAvailable() bool {
	for _, p := range s.providers {
		if p.Available() {
			return true
		}
	}
	return false
}

// AvailableProviders returns list of available provider names
func (s *Service) AvailableProviders() []string {
	var names []string
	for _, p := range s.providers {
		if p.Available() {
			names = append(names, p.Name())
		}
	}
	return names
}

// Helper functions

func numericToFloat64(n pgtype.Numeric) float64 {
	f, _ := n.Float64Value()
	return f.Float64
}

func float64ToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

func strPtr(s string) *string {
	return &s
}
