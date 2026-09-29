-- name: ListCategories :many
SELECT slug, label_id, shortcut_order, group_slug
FROM categories
WHERE publication_status = 'published'
ORDER BY shortcut_order, slug;

-- name: ListCategoryGroups :many
SELECT slug, label_id, shortcut_order
FROM category_groups
WHERE publication_status = 'published'
ORDER BY shortcut_order, slug;

-- name: ListAllCategories :many
SELECT slug, label_id, shortcut_order, group_slug, publication_status, source, proposed_by_maker_id
FROM categories
ORDER BY shortcut_order, slug;

-- name: GetCategory :one
SELECT slug, label_id, shortcut_order, group_slug, publication_status, source, proposed_by_maker_id
FROM categories
WHERE slug = $1;

-- name: GetCategoryGroup :one
SELECT slug, label_id, shortcut_order, publication_status, source, proposed_by_maker_id
FROM category_groups
WHERE slug = $1;

-- name: GetCategoryByLabel :one
SELECT slug, label_id
FROM categories
WHERE lower(label_id) = lower($1);

-- name: GetCategoryGroupByLabel :one
SELECT slug, label_id
FROM category_groups
WHERE lower(label_id) = lower($1);

-- name: InsertCategoryGroup :one
INSERT INTO category_groups (slug, label_id, shortcut_order, publication_status, source, proposed_by_maker_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: InsertCategoryRow :one
INSERT INTO categories (slug, label_id, shortcut_order, group_slug, publication_status, source, proposed_by_maker_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: SetCategoryPublication :exec
UPDATE categories
SET publication_status = $2
WHERE slug = $1;

-- name: SetCategoryGroupPublication :exec
UPDATE category_groups
SET publication_status = $2
WHERE slug = $1;

-- name: ListDraftCategories :many
SELECT slug, label_id, shortcut_order, group_slug, publication_status, source, proposed_by_maker_id
FROM categories
WHERE publication_status = 'draft'
ORDER BY slug;

-- name: ListDraftCategoryGroups :many
SELECT slug, label_id, shortcut_order, publication_status, source, proposed_by_maker_id, created_at
FROM category_groups
WHERE publication_status = 'draft'
ORDER BY created_at DESC;

-- name: ListMakerDraftCategories :many
SELECT slug, label_id, shortcut_order, group_slug, publication_status
FROM categories
WHERE proposed_by_maker_id = $1
ORDER BY slug;

-- name: ListMakerDraftCategoryGroups :many
SELECT slug, label_id, shortcut_order, publication_status
FROM category_groups
WHERE proposed_by_maker_id = $1
ORDER BY slug;

-- name: InsertSynonym :exec
INSERT INTO category_synonyms (category_slug, synonym)
VALUES ($1, $2)
ON CONFLICT (synonym) DO NOTHING;

-- name: CountPublishedMakers :one
SELECT count(*)::bigint
FROM makers
WHERE publication_status = 'published'
  AND is_demo_fixture = FALSE;

-- name: GetSynonym :one
SELECT category_slug, synonym
FROM category_synonyms
WHERE synonym = $1;

-- name: ListPublishedMakers :many
SELECT
    m.id,
    m.name,
    m.city,
    m.verification_date,
    m.moq_quantity,
    m.moq_basis,
    m.moq_basis_note,
    m.lead_time_estimate,
    m.price_min_idr,
    m.price_max_idr,
    m.price_note
FROM makers m
WHERE m.publication_status = 'published'
  AND m.is_demo_fixture = FALSE
  AND (
      sqlc.narg('category')::text IS NULL
      OR EXISTS (
          SELECT 1 FROM maker_capabilities c
          WHERE c.maker_id = m.id AND c.category_slug = sqlc.narg('category')
      )
  )
ORDER BY m.name;

-- name: ListMakerCapabilities :many
SELECT mc.maker_id, mc.category_slug, cat.label_id, cat.publication_status
FROM maker_capabilities mc
JOIN categories cat ON cat.slug = mc.category_slug
WHERE mc.maker_id = ANY (sqlc.arg('maker_ids')::uuid[]);

-- name: GetMaker :one
SELECT
    m.id,
    m.name,
    m.city,
    m.publication_status,
    m.verification_date,
    m.moq_quantity,
    m.moq_basis,
    m.moq_basis_note,
    m.lead_time_estimate,
    m.price_min_idr,
    m.price_max_idr,
    m.price_note,
    m.conditions,
    m.is_demo_fixture,
    m.enmasse_vendor_id,
    m.enmasse_user_id,
    m.hours_open,
    m.hours_close,
    m.hours_days,
    m.service_area
FROM makers m
WHERE m.id = $1;

-- name: GetMakerByEnmasseVendor :one
SELECT
    m.id,
    m.name,
    m.city,
    m.publication_status,
    m.verification_date,
    m.moq_quantity,
    m.moq_basis,
    m.moq_basis_note,
    m.lead_time_estimate,
    m.price_min_idr,
    m.price_max_idr,
    m.price_note,
    m.conditions,
    m.is_demo_fixture,
    m.enmasse_vendor_id,
    m.enmasse_user_id,
    m.hours_open,
    m.hours_close,
    m.hours_days,
    m.service_area
FROM makers m
WHERE m.enmasse_vendor_id = $1;

-- name: UpdateMakerOverlay :exec
UPDATE makers
SET hours_open = $2,
    hours_close = $3,
    hours_days = $4,
    price_min_idr = $5,
    price_max_idr = $6,
    price_note = $7,
    moq_quantity = $8,
    moq_basis = $9,
    moq_basis_note = $10,
    lead_time_estimate = $11,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateMakerIdentity :exec
UPDATE makers
SET name = $2,
    city = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: ListMakerMedia :many
SELECT id, maker_id, url, alt, sort_order
FROM maker_media
WHERE maker_id = $1
ORDER BY sort_order, id;

-- name: DeleteMakerMedia :exec
DELETE FROM maker_media WHERE maker_id = $1;

-- name: InsertMakerMedia :exec
INSERT INTO maker_media (maker_id, url, alt, sort_order)
VALUES ($1, $2, $3, $4);

-- name: GetMakerContact :one
SELECT id, maker_id, channel, e164_digits, publish_permitted, permission_at
FROM maker_contacts
WHERE maker_id = $1;

-- name: ListPublishedRecommendations :many
SELECT
    r.id,
    r.product_made,
    r.production_date_approx,
    r.body,
    r.quantity,
    r.would_use_again,
    a.display_name
FROM recommendations r
JOIN artists a ON a.id = r.artist_id
WHERE r.maker_id = $1
  AND r.publication_status = 'published'
ORDER BY r.created_at DESC;
