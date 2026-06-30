-- Seed Data for Quick Commerce
-- Run: mysql -u root -p quickcommerce < migrations/002_seed_data.sql
-- Timestamps: epoch milliseconds (June 2026)

USE quickcommerce;

-- ============================================================
-- USERS
-- ============================================================
INSERT INTO users (id, contact, email, status, created_at, updated_at) VALUES
('user-001', '9876543210', 'alice@example.com',   'active',   1782700000000, 1782700000000),
('user-002', '9876543211', 'bob@example.com',     'active',   1782700000000, 1782700000000),
('user-003', '9876543212', 'charlie@example.com', 'inactive', 1782700000000, 1782700000000),
('user-004', '9876543213', 'diana@example.com',   'active',   1782700000000, 1782700000000),
('user-005', '9876543214', 'eve@example.com',     'created',  1782700000000, 1782700000000);

-- ============================================================
-- PRODUCTS (amount in paise: 2000 = ₹20.00)
-- ============================================================
INSERT INTO products (id, description, brand, amount, quantity, metadata, created_at, updated_at) VALUES
('prod-001', '45gms',               'Lays Classic Salted',       2000,  100, '{"prev_amount": 1500}', 1782700000000, 1782700000000),
('prod-002', '500ml',               'Coca Cola',                 4000,   50, '{"prev_amount": 3500}', 1782700000000, 1782700000000),
('prod-003', '1L',                  'Amul Toned Milk',           6800,  200, '{"prev_amount": 6000}', 1782700000000, 1782700000000),
('prod-004', '200gms',              'Uncle Chips Spicy',         3000,   80, '{"prev_amount": 2500}', 1782700000000, 1782700000000),
('prod-005', '100gms pack of 4',    'Maggi 2-Minute Noodles',    1400,  150, '{"prev_amount": 1200}', 1782700000000, 1782700000000),
('prod-006', '750ml',               'Pepsi',                     3800,   60, '{"prev_amount": 3200}', 1782700000000, 1782700000000),
('prod-007', '400gms',              'Britannia Good Day',        5500,   90, '{"prev_amount": 5000}', 1782700000000, 1782700000000),
('prod-008', '1kg',                 'Aashirvaad Atta',          18000,  120, '{"prev_amount": 16000}',1782700000000, 1782700000000),
('prod-009', '200ml',               'Paper Boat Aam Panna',      3000,   70, '{"prev_amount": 2500}', 1782700000000, 1782700000000),
('prod-010', '500gms',              'Haldiram Aloo Bhujia',      9500,   40, '{"prev_amount": 8500}', 1782700000000, 1782700000000);
