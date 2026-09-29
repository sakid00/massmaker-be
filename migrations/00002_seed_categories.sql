-- +goose Up
INSERT INTO categories (slug, label_id, shortcut_order) VALUES
    ('sticker', 'Sticker', 1),
    ('artprint', 'Artprint', 2),
    ('print-3d', '3D Print', 3),
    ('pin', 'Pin', 4),
    ('kaos', 'Kaos', 5),
    ('totebag', 'Totebag', 6),
    ('topi', 'Topi', 7),
    ('gantungan', 'Gantungan', 8);

INSERT INTO category_synonyms (category_slug, synonym) VALUES
    ('sticker', 'sticker'),
    ('sticker', 'stiker'),
    ('artprint', 'artprint'),
    ('artprint', 'art print'),
    ('print-3d', '3d print'),
    ('print-3d', '3d'),
    ('pin', 'pin'),
    ('kaos', 'kaos'),
    ('kaos', 'tshirt'),
    ('kaos', 't-shirt'),
    ('totebag', 'totebag'),
    ('totebag', 'tote'),
    ('topi', 'topi'),
    ('topi', 'hat'),
    ('gantungan', 'gantungan'),
    ('gantungan', 'keychain');

-- +goose Down
DELETE FROM category_synonyms;
DELETE FROM categories;
