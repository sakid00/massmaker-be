package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type makerSeed struct {
	ID         uuid.UUID
	Name       string
	City       string
	Verified   string
	Category   string
	MOQ        int32
	Basis      string
	Lead       string
	MinIDR     int32
	MaxIDR     int32
	PriceNote  string
	Conditions string
	Service    string
}

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "seeddev: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	mmURL, emURL, err := loadURLs()
	if err != nil {
		return err
	}
	mm, err := pgx.Connect(ctx, mmURL)
	if err != nil {
		return fmt.Errorf("massmaker: %w", err)
	}
	defer mm.Close(ctx)
	em, err := pgx.Connect(ctx, emURL)
	if err != nil {
		return fmt.Errorf("enmasse: %w", err)
	}
	defer em.Close(ctx)

	makers := []makerSeed{
		{
			ID:   uuid.MustParse("00000000-0000-4000-8000-000000000001"),
			Name: "Stiker Rumah", City: "Jakarta", Verified: "2026-08-01",
			Category: "sticker", MOQ: 100, Basis: "per_desain", Lead: "5 hari",
			MinIDR: 3000, MaxIDR: 8000, PriceNote: "per pcs vinyl",
			Conditions: "Vinyl tahan air.", Service: "Jabodetabek",
		},
		{
			ID:   uuid.MustParse("00000000-0000-4000-8000-000000000002"),
			Name: "Sablon Selatan", City: "Jakarta Selatan", Verified: "2026-07-20",
			Category: "kaos", MOQ: 24, Basis: "per_desain", Lead: "10 hari",
			MinIDR: 55000, MaxIDR: 90000, PriceNote: "cotton combed 24s",
			Conditions: "Desain final dalam format vektor.", Service: "Jabodetabek",
		},
		{
			ID:   uuid.MustParse("00000000-0000-4000-8000-000000000003"),
			Name: "Karya Topi Bandung", City: "Bandung", Verified: "2026-08-12",
			Category: "topi", MOQ: 50, Basis: "per_desain", Lead: "14 hari",
			MinIDR: 45000, MaxIDR: 75000, PriceNote: "per pcs, tergantung bahan",
			Conditions: "Desain final dalam format vektor.", Service: "Jawa Barat",
		},
		{
			ID:   uuid.MustParse("00000000-0000-4000-8000-000000000004"),
			Name: "Ganci Jogja", City: "Yogyakarta", Verified: "2026-06-02",
			Category: "gantungan", MOQ: 50, Basis: "total", Lead: "7 hari",
			MinIDR: 8000, MaxIDR: 15000, PriceNote: "akrilik 3mm",
			Conditions: "Akrilik 3mm default.", Service: "DI Yogyakarta",
		},
	}

	for _, m := range makers {
		if err := upsertMaker(ctx, mm, m); err != nil {
			return fmt.Errorf("maker %s: %w", m.Name, err)
		}
		fmt.Printf("published maker %s %s (%s)\n", m.ID, m.Name, m.Category)
	}

	vendorUserID, vendorID, vendorEmail, createdVendor, err := ensureVendor(ctx, em)
	if err != nil {
		return err
	}
	if createdVendor {
		fmt.Printf("created vendor %s password=seed-dev-pass\n", vendorEmail)
	} else {
		fmt.Printf("using vendor %s\n", vendorEmail)
	}

	boundID, err := bindVendorMaker(ctx, mm, makers[0].ID, vendorUserID, vendorID)
	if err != nil {
		return err
	}
	fmt.Printf("vendor inbox bound to maker %s\n", boundID)

	artistUserID, artistID, artistName, artistEmail, err := firstArtist(ctx, em)
	if err != nil {
		fmt.Printf("no artist yet; skip sample order (%v)\n", err)
		return nil
	}
	if err := upsertSampleOrder(ctx, mm, boundID, makers, artistUserID, artistID, artistName); err != nil {
		return err
	}
	fmt.Printf("sample pending order for artist %s (%s)\n", artistEmail, artistName)
	return nil
}

func loadURLs() (string, string, error) {
	_ = godotenv.Load(".env")
	mm := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if mm == "" {
		return "", "", fmt.Errorf("DATABASE_URL missing")
	}
	emFile := "../enmasse-be/.env"
	if _, err := os.Stat(emFile); err != nil {
		emFile = os.Getenv("ENMASSE_ENV")
	}
	emMap, err := godotenv.Read(emFile)
	if err != nil {
		return "", "", fmt.Errorf("enmasse env: %w", err)
	}
	em := strings.TrimSpace(emMap["DATABASE_URL"])
	if em == "" {
		return "", "", fmt.Errorf("enmasse DATABASE_URL missing")
	}
	return mm, em, nil
}

func upsertMaker(ctx context.Context, db *pgx.Conn, m makerSeed) error {
	_, err := db.Exec(ctx, `
INSERT INTO makers (
    id, name, city, publication_status, verification_date,
    moq_quantity, moq_basis, lead_time_estimate,
    price_min_idr, price_max_idr, price_note, conditions,
    is_demo_fixture, hours_open, hours_close, hours_days, service_area
) VALUES (
    $1, $2, $3, 'published', $4::date,
    $5, $6, $7,
    $8, $9, $10, $11,
    FALSE, '09:00', '17:00', ARRAY['mon','tue','wed','thu','fri']::text[], $12
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    city = EXCLUDED.city,
    publication_status = 'published',
    verification_date = EXCLUDED.verification_date,
    moq_quantity = EXCLUDED.moq_quantity,
    moq_basis = EXCLUDED.moq_basis,
    lead_time_estimate = EXCLUDED.lead_time_estimate,
    price_min_idr = EXCLUDED.price_min_idr,
    price_max_idr = EXCLUDED.price_max_idr,
    price_note = EXCLUDED.price_note,
    conditions = EXCLUDED.conditions,
    is_demo_fixture = FALSE,
    hours_open = EXCLUDED.hours_open,
    hours_close = EXCLUDED.hours_close,
    hours_days = EXCLUDED.hours_days,
    service_area = EXCLUDED.service_area,
    updated_at = NOW()`,
		m.ID, m.Name, m.City, m.Verified, m.MOQ, m.Basis, m.Lead,
		m.MinIDR, m.MaxIDR, m.PriceNote, m.Conditions, m.Service,
	)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `
INSERT INTO maker_capabilities (maker_id, category_slug)
VALUES ($1, $2)
ON CONFLICT DO NOTHING`, m.ID, m.Category)
	return err
}

func ensureVendor(ctx context.Context, db *pgx.Conn) (userID, vendorID uuid.UUID, email string, created bool, err error) {
	err = db.QueryRow(ctx, `
SELECT u.id, v.id, u.email::text
FROM users u
JOIN vendors v ON v.user_id = u.id
WHERE u.role = 'vendor'
ORDER BY u.created_at
LIMIT 1`).Scan(&userID, &vendorID, &email)
	if err == nil {
		if err := completeVendor(ctx, db, vendorID); err != nil {
			return uuid.Nil, uuid.Nil, "", false, err
		}
		return userID, vendorID, email, false, nil
	}
	if err != pgx.ErrNoRows {
		return uuid.Nil, uuid.Nil, "", false, fmt.Errorf("list vendors: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("seed-dev-pass"), bcrypt.DefaultCost)
	if err != nil {
		return uuid.Nil, uuid.Nil, "", false, err
	}
	email = "vendor.seed@massmaker.local"
	err = db.QueryRow(ctx, `
INSERT INTO users (email, password_hash, role, handle, must_change_password)
VALUES ($1, $2, 'vendor', 'seed-stiker-rumah', FALSE)
ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash
RETURNING id`, email, string(hash)).Scan(&userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, "", false, fmt.Errorf("insert vendor user: %w", err)
	}
	err = db.QueryRow(ctx, `
INSERT INTO vendors (
    user_id, name, city, pic_name, whatsapp, address,
    province, province_code, city_code, district, district_code, bio
) VALUES (
    $1, 'Stiker Rumah', 'Jakarta', 'Budi Seed', '+6281110001004',
    'Jl. Seed No. 1', 'DKI Jakarta', '31', '3171', 'Gambir', '317101',
    'Vendor seed untuk uji inquiry dan order.'
)
ON CONFLICT (user_id) DO UPDATE SET
    name = EXCLUDED.name,
    city = EXCLUDED.city,
    pic_name = EXCLUDED.pic_name,
    whatsapp = EXCLUDED.whatsapp,
    address = EXCLUDED.address,
    province = EXCLUDED.province,
    district = EXCLUDED.district,
    bio = EXCLUDED.bio
RETURNING id`, userID).Scan(&vendorID)
	if err != nil {
		return uuid.Nil, uuid.Nil, "", false, fmt.Errorf("insert vendor: %w", err)
	}
	return userID, vendorID, email, true, nil
}

func completeVendor(ctx context.Context, db *pgx.Conn, vendorID uuid.UUID) error {
	_, err := db.Exec(ctx, `
UPDATE vendors SET
    name = CASE WHEN btrim(coalesce(name, '')) = '' THEN 'Stiker Rumah' ELSE name END,
    city = CASE WHEN btrim(coalesce(city, '')) = '' THEN 'Jakarta' ELSE city END,
    pic_name = CASE WHEN btrim(coalesce(pic_name, '')) = '' THEN 'Budi Seed' ELSE pic_name END,
    whatsapp = CASE WHEN btrim(coalesce(whatsapp, '')) = '' THEN '+6281110001004' ELSE whatsapp END,
    address = CASE WHEN btrim(coalesce(address, '')) = '' THEN 'Jl. Seed No. 1' ELSE address END,
    province = CASE WHEN btrim(coalesce(province, '')) = '' THEN 'DKI Jakarta' ELSE province END,
    district = CASE WHEN btrim(coalesce(district, '')) = '' THEN 'Gambir' ELSE district END,
    bio = CASE WHEN btrim(coalesce(bio, '')) = '' THEN 'Vendor seed untuk uji inquiry dan order.' ELSE bio END
WHERE id = $1`, vendorID)
	return err
}

func bindVendorMaker(ctx context.Context, db *pgx.Conn, preferred uuid.UUID, userID, vendorID uuid.UUID) (uuid.UUID, error) {
	var existing uuid.UUID
	err := db.QueryRow(ctx, `SELECT id FROM makers WHERE enmasse_vendor_id = $1 LIMIT 1`, vendorID).Scan(&existing)
	if err == nil {
		_, err = db.Exec(ctx, `
UPDATE makers SET
    publication_status = 'published',
    is_demo_fixture = FALSE,
    enmasse_user_id = $2,
    verification_date = coalesce(verification_date, CURRENT_DATE),
    moq_quantity = coalesce(moq_quantity, 50),
    moq_basis = coalesce(moq_basis, 'per_desain'),
    lead_time_estimate = coalesce(nullif(btrim(lead_time_estimate), ''), '7 hari'),
    price_min_idr = coalesce(price_min_idr, 10000),
    price_max_idr = coalesce(price_max_idr, 25000),
    hours_open = coalesce(nullif(btrim(hours_open), ''), '09:00'),
    hours_close = coalesce(nullif(btrim(hours_close), ''), '17:00'),
    hours_days = CASE WHEN hours_days = '{}' THEN ARRAY['mon','tue','wed','thu','fri']::text[] ELSE hours_days END,
    updated_at = NOW()
WHERE id = $1`, existing, userID)
		if err != nil {
			return uuid.Nil, err
		}
		_, err = db.Exec(ctx, `
INSERT INTO maker_capabilities (maker_id, category_slug)
SELECT $1, 'sticker'
WHERE NOT EXISTS (SELECT 1 FROM maker_capabilities WHERE maker_id = $1)`, existing)
		return existing, err
	}
	if err != pgx.ErrNoRows {
		return uuid.Nil, err
	}
	_, err = db.Exec(ctx, `
UPDATE makers SET
    enmasse_vendor_id = $2,
    enmasse_user_id = $3,
    updated_at = NOW()
WHERE id = $1`, preferred, vendorID, userID)
	return preferred, err
}

func firstArtist(ctx context.Context, db *pgx.Conn) (userID, artistID uuid.UUID, name, email string, err error) {
	var artistName, firstName *string
	err = db.QueryRow(ctx, `
SELECT u.id, a.id, u.email::text, nullif(btrim(a.artist_name), ''), nullif(btrim(a.first_name), '')
FROM users u
JOIN artists a ON a.user_id = u.id
WHERE u.role = 'artist'
ORDER BY u.created_at DESC
LIMIT 1`).Scan(&userID, &artistID, &email, &artistName, &firstName)
	if err != nil {
		return uuid.Nil, uuid.Nil, "", "", err
	}
	name = email
	if firstName != nil && *firstName != "" {
		name = *firstName
	}
	if artistName != nil && *artistName != "" {
		name = *artistName
	}
	return userID, artistID, name, email, nil
}

func upsertSampleOrder(ctx context.Context, db *pgx.Conn, makerID uuid.UUID, makers []makerSeed, artistUserID, artistID uuid.UUID, artistName string) error {
	maker := makers[0]
	for _, item := range makers {
		if item.ID == makerID {
			maker = item
			break
		}
	}
	var category string
	err := db.QueryRow(ctx, `
SELECT category_slug FROM maker_capabilities WHERE maker_id = $1 LIMIT 1`, makerID).Scan(&category)
	if err != nil {
		category = maker.Category
	}
	ready := time.Now().AddDate(0, 0, 14).Format("2006-01-02")
	ship := time.Now().AddDate(0, 0, 21).Format("2006-01-02")
	_, err = db.Exec(ctx, `
INSERT INTO orders (
    reference_code, maker_id, enmasse_user_id, enmasse_artist_id,
    artist_display_name, maker_name, maker_city, status,
    project_name, product_name, category_slug, quantity, design_count,
    want_sample, requested_ready_on, requested_ship_on,
    ship_to_province, ship_to_city, ship_to_district, notes
) VALUES (
    'MM-SEED-001', $1, $2, $3,
    $4, $5, $6, 'pending_review',
    'Booth merch JICAF', 'Stiker vinyl A6', $7, 200, 2,
    FALSE, $8::date, $9::date,
    'DKI Jakarta', 'Jakarta', 'Gambir', 'Order seed untuk uji tinjauan vendor.'
)
ON CONFLICT (reference_code) DO NOTHING`,
		makerID, artistUserID, artistID, artistName, maker.Name, maker.City, category, ready, ship,
	)
	return err
}
