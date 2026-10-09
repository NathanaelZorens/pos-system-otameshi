CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('waiter', 'admin')),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS members (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS dining_tables (
    id TEXT PRIMARY KEY,
    label TEXT NOT NULL UNIQUE,
    capacity INTEGER NOT NULL DEFAULT 4,
    is_active INTEGER NOT NULL DEFAULT 1,
    guest_token TEXT UNIQUE
);

CREATE TABLE IF NOT EXISTS menu_categories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS menu_items (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    price_cents INTEGER NOT NULL,
    category TEXT,
    category_id TEXT REFERENCES menu_categories(id),
    is_sold_out INTEGER NOT NULL DEFAULT 0,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    dining_table_id TEXT REFERENCES dining_tables(id),
    waiter_id TEXT REFERENCES users(id),
    member_id TEXT REFERENCES members(id),
    source TEXT NOT NULL DEFAULT 'staff' CHECK (source IN ('staff', 'guest', 'takeout')),
    status TEXT NOT NULL CHECK (status IN ('open', 'preparing', 'served', 'ready', 'completed', 'paid', 'cleared', 'cancelled')) DEFAULT 'open',
    customer_name TEXT,
    guest_claim TEXT,
    cancel_reason TEXT,
    pickup_code TEXT,
    total_cents INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    cancelled_at TEXT,
    served_at TEXT
);

CREATE TABLE IF NOT EXISTS order_items (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    menu_item_id TEXT NOT NULL REFERENCES menu_items(id),
    name_snapshot TEXT NOT NULL,
    unit_price_cents INTEGER NOT NULL,
    list_unit_price_cents INTEGER,
    discount_rule_id TEXT,
    discount_label TEXT,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed'))
);

CREATE TABLE IF NOT EXISTS discount_rules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    target_type TEXT NOT NULL CHECK (target_type IN ('item', 'category')),
    target_id TEXT NOT NULL,
    discount_type TEXT NOT NULL CHECK (discount_type IN ('fixed', 'percent')),
    amount INTEGER NOT NULL CHECK (amount > 0),
    start_time TEXT,
    end_time TEXT,
    weekdays TEXT,
    starts_on TEXT,
    ends_on TEXT,
    is_active INTEGER NOT NULL DEFAULT 1,
    is_featured_price INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_discount_rules_target ON discount_rules(target_type, target_id);

CREATE TABLE IF NOT EXISTS payments (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL UNIQUE REFERENCES orders(id),
    method TEXT NOT NULL CHECK (method IN ('cash', 'card', 'qr', 'wallet')),
    amount_cents INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'succeeded',
    provider TEXT NOT NULL,
    provider_ref TEXT,
    paid_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_menu_item_id ON order_items(menu_item_id);
