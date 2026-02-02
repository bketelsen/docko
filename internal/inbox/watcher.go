package inbox

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// debouncer handles debouncing file events to avoid processing the same file
// multiple times when it's being written in chunks.
type debouncer struct {
	timers map[string]*time.Timer
	mu     sync.Mutex
}

// newDebouncer creates a new debouncer instance.
func newDebouncer() *debouncer {
	return &debouncer{
		timers: make(map[string]*time.Timer),
	}
}

// Debounce schedules fn to be called after delay for the given path.
// If called again for the same path before delay expires, the timer resets.
func (d *debouncer) Debounce(path string, delay time.Duration, fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Cancel existing timer for this path
	if timer, ok := d.timers[path]; ok {
		timer.Stop()
	}

	// Schedule new timer
	d.timers[path] = time.AfterFunc(delay, func() {
		d.mu.Lock()
		delete(d.timers, path)
		d.mu.Unlock()
		fn()
	})
}

// Cancel stops the timer for the given path without calling the handler.
func (d *debouncer) Cancel(path string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if timer, ok := d.timers[path]; ok {
		timer.Stop()
		delete(d.timers, path)
	}
}

// CancelAll stops all pending timers.
func (d *debouncer) CancelAll() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for path, timer := range d.timers {
		timer.Stop()
		delete(d.timers, path)
	}
}

// Watcher watches directories for new PDF files using fsnotify.
type Watcher struct {
	watcher   *fsnotify.Watcher
	debouncer *debouncer
	handler   func(path string) // Called for each stable file
	delay     time.Duration     // Debounce delay (default 500ms)
	mu        sync.RWMutex
	watching  map[string]struct{} // Tracked directories
}

// NewWatcher creates a new file watcher with the given debounce delay and handler.
// The handler is called for each file after no events have been received for the delay duration.
func NewWatcher(delay time.Duration, handler func(string)) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if delay == 0 {
		delay = 500 * time.Millisecond
	}

	return &Watcher{
		watcher:   fsWatcher,
		debouncer: newDebouncer(),
		handler:   handler,
		delay:     delay,
		watching:  make(map[string]struct{}),
	}, nil
}

// Add starts watching the given directory for file changes.
func (w *Watcher) Add(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.watching[path]; exists {
		return nil // Already watching
	}

	if err := w.watcher.Add(path); err != nil {
		return err
	}

	w.watching[path] = struct{}{}
	slog.Info("watching directory", "path", path)
	return nil
}

// Remove stops watching the given directory.
func (w *Watcher) Remove(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.watching[path]; !exists {
		return nil // Not watching
	}

	if err := w.watcher.Remove(path); err != nil {
		return err
	}

	delete(w.watching, path)
	slog.Info("stopped watching directory", "path", path)
	return nil
}

// Run starts the event loop and blocks until context is cancelled.
// It handles Create and Write events, debouncing them before calling the handler.
func (w *Watcher) Run(ctx context.Context) error {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			slog.Error("watcher error", "error", err)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// handleEvent processes a single fsnotify event.
func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Only process Create and Write events
	if !event.Has(fsnotify.Create) && !event.Has(fsnotify.Write) {
		return
	}

	// Only process PDF files
	if !isPDFFilename(event.Name) {
		return
	}

	path := event.Name
	slog.Debug("file event", "op", event.Op.String(), "path", path)

	// Debounce the event - handler will be called after no events for delay duration
	w.debouncer.Debounce(path, w.delay, func() {
		slog.Debug("file stable, processing", "path", path)
		w.handler(path)
	})
}

// Close stops watching all directories and releases resources.
func (w *Watcher) Close() error {
	w.debouncer.CancelAll()
	return w.watcher.Close()
}

// isPDFFilename checks if a filename has a .pdf extension (case-insensitive).
func isPDFFilename(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".pdf")
}
