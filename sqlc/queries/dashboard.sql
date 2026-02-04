-- name: GetDashboardDocumentStats :one
SELECT
    COUNT(*)::int AS total,
    COUNT(*) FILTER (WHERE processing_status = 'completed')::int AS processed,
    COUNT(*) FILTER (WHERE processing_status = 'pending')::int AS pending,
    COUNT(*) FILTER (WHERE processing_status = 'failed')::int AS failed,
    COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE)::int AS today
FROM documents;

-- name: GetDashboardQueueStats :one
SELECT
    COUNT(*) FILTER (WHERE status = 'pending')::int AS pending,
    COUNT(*) FILTER (WHERE status = 'completed')::int AS completed,
    COUNT(*) FILTER (WHERE status = 'failed')::int AS failed,
    COUNT(*) FILTER (WHERE status = 'processing')::int AS processing
FROM jobs;

-- name: GetDashboardSourceStats :one
SELECT
    (SELECT COUNT(*)::int FROM inboxes) AS inbox_total,
    (SELECT COUNT(*)::int FROM inboxes WHERE enabled = true) AS inbox_enabled,
    (SELECT COUNT(*)::int FROM network_sources) AS network_total,
    (SELECT COUNT(*)::int FROM network_sources WHERE enabled = true) AS network_enabled;

-- name: CountTags :one
SELECT COUNT(*)::int FROM tags;

-- name: CountCorrespondents :one
SELECT COUNT(*)::int FROM correspondents;

-- name: GetDashboardJobsToday :one
SELECT COUNT(*)::int FROM jobs WHERE created_at >= CURRENT_DATE;
