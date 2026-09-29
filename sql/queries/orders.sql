-- name: InsertOrder :one
INSERT INTO orders (
    reference_code,
    maker_id,
    enmasse_user_id,
    enmasse_artist_id,
    artist_display_name,
    maker_name,
    maker_city,
    project_name,
    product_name,
    category_slug,
    quantity,
    quantity_note,
    design_count,
    want_sample,
    sample_quantity,
    requested_ready_on,
    requested_ship_on,
    rush,
    budget_min_idr,
    budget_max_idr,
    notes,
    ship_to_province,
    ship_to_city,
    ship_to_district,
    ship_to_address
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
    $21, $22, $23, $24, $25
)
RETURNING *;

-- name: InsertOrderAttachment :one
INSERT INTO order_attachments (
    order_id, url, content_type, original_filename, byte_size, sort_order
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetOrder :one
SELECT * FROM orders WHERE id = $1;

-- name: ListOrdersByArtist :many
SELECT * FROM orders
WHERE enmasse_user_id = $1
ORDER BY created_at DESC;

-- name: ListOrdersByMaker :many
SELECT * FROM orders
WHERE maker_id = $1
ORDER BY created_at DESC;

-- name: ListOrderAttachments :many
SELECT * FROM order_attachments
WHERE order_id = $1
ORDER BY sort_order, id;

-- name: ListOrderAttachmentsForOrders :many
SELECT * FROM order_attachments
WHERE order_id = ANY (sqlc.arg('order_ids')::uuid[])
ORDER BY sort_order, id;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $2,
    vendor_note = $3,
    status_changed_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetMakerByEnmasseUser :one
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
WHERE m.enmasse_user_id = $1
LIMIT 1;
