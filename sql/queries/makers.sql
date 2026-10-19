-- name: InsertMaker :one
INSERT INTO makers (
    name, city, verification_date, moq_quantity, moq_basis, moq_basis_note,
    lead_time_estimate, price_min_idr, price_max_idr, price_note, conditions,
    enmasse_vendor_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: ReplaceCapabilities :exec
DELETE FROM maker_capabilities WHERE maker_id = $1;

-- name: InsertCapability :exec
INSERT INTO maker_capabilities (maker_id, category_slug)
VALUES ($1, $2);

-- name: UpsertContact :one
INSERT INTO maker_contacts (maker_id, channel, e164_digits, publish_permitted)
VALUES ($1, $2, $3, FALSE)
ON CONFLICT (maker_id) DO UPDATE
SET channel = EXCLUDED.channel, e164_digits = EXCLUDED.e164_digits
RETURNING *;

-- name: SetMakerEnmasseUserID :exec
UPDATE makers
SET enmasse_user_id = $2, updated_at = NOW()
WHERE id = $1;
