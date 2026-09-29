-- name: GetConsentBySessionType :one
SELECT *
FROM consents
WHERE anon_session_id = $1
  AND consent_type = $2;

-- name: GetConsentByUserType :one
SELECT *
FROM consents
WHERE user_id = $1
  AND consent_type = $2;

-- name: ListConsentsByUser :many
SELECT *
FROM consents
WHERE user_id = $1
ORDER BY consent_type;

-- name: InsertConsent :one
INSERT INTO consents (
    user_id, anon_session_id, consent_type, version, granted,
    granted_at, withdrawn_at, source, ip_hash, user_agent_hash
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: UpdateConsent :one
UPDATE consents
SET
    user_id = COALESCE($2, user_id),
    anon_session_id = $3,
    version = $4,
    granted = $5,
    granted_at = $6,
    withdrawn_at = $7,
    source = $8,
    ip_hash = $9,
    user_agent_hash = $10,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: BackfillConsentUser :exec
UPDATE consents
SET user_id = $2, updated_at = NOW()
WHERE consents.anon_session_id = $1
  AND consents.user_id IS NULL
  AND NOT EXISTS (
      SELECT 1
      FROM consents other
      WHERE other.user_id = $2
        AND other.consent_type = consents.consent_type
  );

-- name: ActivityTrackingGranted :one
SELECT EXISTS (
    SELECT 1
    FROM consents
    WHERE consent_type = 'activity_tracking'
      AND granted = TRUE
      AND withdrawn_at IS NULL
      AND (
          anon_session_id = sqlc.arg('anon_session_id')
          OR (sqlc.narg('user_id')::uuid IS NOT NULL AND user_id = sqlc.narg('user_id'))
      )
)::bool AS granted;

-- name: WhatsAppPublicationGrantedForMaker :one
SELECT EXISTS (
    SELECT 1
    FROM consents c
    JOIN makers m ON m.enmasse_user_id = c.user_id
    WHERE m.id = $1
      AND c.consent_type = 'whatsapp_publication'
      AND c.granted = TRUE
      AND c.withdrawn_at IS NULL
)::bool AS granted;
