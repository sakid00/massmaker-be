-- name: GetStaffByEmail :one
SELECT id, email, password_hash, role, created_at
FROM staff_users
WHERE email = $1;

-- name: CountStaff :one
SELECT count(*)::bigint FROM staff_users;

-- name: InsertStaff :one
INSERT INTO staff_users (email, password_hash, role)
VALUES ($1, $2, $3)
RETURNING *;

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

-- name: SetPublication :exec
UPDATE makers
SET publication_status = $2, updated_at = NOW()
WHERE id = $1;

-- name: UpsertContact :one
INSERT INTO maker_contacts (maker_id, channel, e164_digits, publish_permitted)
VALUES ($1, $2, $3, FALSE)
ON CONFLICT (maker_id) DO UPDATE
SET channel = EXCLUDED.channel, e164_digits = EXCLUDED.e164_digits
RETURNING *;

-- name: PermitContact :exec
UPDATE maker_contacts
SET publish_permitted = TRUE, permission_at = NOW()
WHERE maker_id = $1;

-- name: SetMakerEnmasseUserID :exec
UPDATE makers
SET enmasse_user_id = $2, updated_at = NOW()
WHERE id = $1;

-- name: ListAdminMakers :many
SELECT id, name, city, publication_status, enmasse_vendor_id, created_at
FROM makers
WHERE is_demo_fixture = FALSE
ORDER BY created_at DESC;
