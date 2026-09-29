-- name: InsertEvent :one
INSERT INTO events (
    type, traffic_source, anon_session_id, enmasse_user_id, raw_query, interpreted_category,
    matched_synonym, result_count, no_result_reason, maker_id, counted
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: CountRecentContactClicks :one
SELECT COUNT(*)::int AS n
FROM events
WHERE type = 'contact_click'
  AND counted = TRUE
  AND anon_session_id = $1
  AND maker_id = $2
  AND created_at >= $3;

-- name: UpsertSession :exec
INSERT INTO sessions (anon_session_id, enmasse_user_id, last_flush_at)
VALUES ($1, $2, NOW())
ON CONFLICT (anon_session_id) DO UPDATE SET
    last_flush_at = NOW(),
    enmasse_user_id = COALESCE(sessions.enmasse_user_id, EXCLUDED.enmasse_user_id);

-- name: BackfillSessionUser :exec
UPDATE events
SET enmasse_user_id = $2
WHERE anon_session_id = $1
  AND enmasse_user_id IS NULL;
