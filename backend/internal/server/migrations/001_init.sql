CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE users (
    id                      BIGSERIAL PRIMARY KEY,
    role                    TEXT NOT NULL CHECK (role IN ('customer', 'technician', 'seller', 'admin')),
    name                    TEXT NOT NULL,
    email                   TEXT NOT NULL UNIQUE,
    phone                   TEXT NOT NULL,
    password_hash           TEXT NOT NULL,
    balance_tetri           BIGINT NOT NULL DEFAULT 0 CHECK (balance_tetri >= 0),
    terms_version           TEXT NOT NULL,
    terms_accepted_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    registration_paid_until TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE categories (
    id              BIGSERIAL PRIMARY KEY,
    kind            TEXT NOT NULL CHECK (kind IN ('repair', 'part')),
    slug            TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    labor_min_tetri BIGINT NOT NULL DEFAULT 0,
    labor_max_tetri BIGINT NOT NULL DEFAULT 0,
    parts_min_tetri BIGINT NOT NULL DEFAULT 0,
    parts_max_tetri BIGINT NOT NULL DEFAULT 0
);

INSERT INTO categories (kind, slug, name, labor_min_tetri, labor_max_tetri, parts_min_tetri, parts_max_tetri) VALUES
    ('repair', 'air-conditioner',  'Air conditioner',          8000, 20000, 5000, 40000),
    ('repair', 'refrigerator',     'Refrigerator',             6000, 15000, 5000, 35000),
    ('repair', 'washing-machine',  'Washing machine',          5000, 15000, 4000, 30000),
    ('repair', 'dishwasher',       'Dishwasher',               5000, 14000, 4000, 25000),
    ('repair', 'tv-electronics',   'TV & electronics',         4000, 12000, 3000, 30000),
    ('repair', 'small-appliances', 'Small appliances',         2000,  6000, 1000,  8000),
    ('repair', 'phone-computer',   'Phones & computers',       3000, 10000, 2000, 30000),
    ('repair', 'electrical',       'Electrical wiring',        4000, 15000, 2000, 15000),
    ('repair', 'plumbing',         'Plumbing',                 4000, 15000, 2000, 20000),
    ('part',   'large-gadgets',    'Large gadgets (fridge, washer, AC)', 0, 0, 0, 0),
    ('part',   'small-gadgets',    'Small gadgets (hand mixer, kettle, iron)', 0, 0, 0, 0),
    ('part',   'ac-parts',         'AC parts', 0, 0, 0, 0),
    ('part',   'fridge-parts',     'Fridge & freezer parts', 0, 0, 0, 0),
    ('part',   'washer-parts',     'Washing machine parts', 0, 0, 0, 0),
    ('part',   'electronic-components', 'Electronic components', 0, 0, 0, 0),
    ('part',   'tools',            'Tools', 0, 0, 0, 0);

CREATE TABLE technician_profiles (
    user_id           BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    bio               TEXT NOT NULL DEFAULT '',
    city              TEXT NOT NULL DEFAULT '',
    location          GEOGRAPHY(POINT, 4326),
    service_radius_km INT NOT NULL DEFAULT 10 CHECK (service_radius_km BETWEEN 1 AND 500),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX technician_profiles_location_idx ON technician_profiles USING GIST (location);

CREATE TABLE technician_categories (
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES categories(id),
    PRIMARY KEY (user_id, category_id)
);

CREATE TABLE repair_posts (
    id                 BIGSERIAL PRIMARY KEY,
    customer_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id        BIGINT NOT NULL REFERENCES categories(id),
    title              TEXT NOT NULL,
    description        TEXT NOT NULL,
    address            TEXT NOT NULL DEFAULT '',
    location           GEOGRAPHY(POINT, 4326) NOT NULL,
    has_parts          BOOLEAN NOT NULL DEFAULT false,
    status             TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    estimate_min_tetri BIGINT NOT NULL DEFAULT 0,
    estimate_max_tetri BIGINT NOT NULL DEFAULT 0,
    estimate_note      TEXT NOT NULL DEFAULT '',
    flagged            BOOLEAN NOT NULL DEFAULT false,
    flag_reason        TEXT NOT NULL DEFAULT '',
    duplicate_of       BIGINT REFERENCES repair_posts(id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX repair_posts_location_idx ON repair_posts USING GIST (location);
CREATE INDEX repair_posts_customer_idx ON repair_posts (customer_id, created_at DESC);

CREATE TABLE contact_reveals (
    id            BIGSERIAL PRIMARY KEY,
    technician_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id       BIGINT NOT NULL REFERENCES repair_posts(id) ON DELETE CASCADE,
    price_tetri   BIGINT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (technician_id, post_id)
);

CREATE TABLE stalls (
    id       BIGSERIAL PRIMARY KEY,
    code     TEXT NOT NULL UNIQUE,
    name     TEXT NOT NULL DEFAULT '',
    location GEOGRAPHY(POINT, 4326) NOT NULL
);

-- Placeholder pins around the Eliava market; move them to the real spots from the admin page.
INSERT INTO stalls (code, name, location)
SELECT r || n, 'Row ' || r || ', stall ' || n,
       ST_SetSRID(ST_MakePoint(44.7770 + n * 0.0004, 41.7265 + (ascii(r) - 65) * 0.0004), 4326)::geography
FROM unnest(ARRAY['A', 'B', 'C', 'D']) AS r, generate_series(1, 6) AS n;

CREATE TABLE seller_profiles (
    user_id    BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    shop_name  TEXT NOT NULL DEFAULT '',
    stall_id   BIGINT REFERENCES stalls(id) ON DELETE SET NULL,
    address    TEXT NOT NULL DEFAULT '',
    location   GEOGRAPHY(POINT, 4326),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE listings (
    id          BIGSERIAL PRIMARY KEY,
    seller_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES categories(id),
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price_tetri BIGINT NOT NULL CHECK (price_tetri > 0),
    stock       INT NOT NULL CHECK (stock >= 0),
    active      BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX listings_title_trgm_idx ON listings USING GIN (title gin_trgm_ops);

CREATE TABLE media (
    id         BIGSERIAL PRIMARY KEY,
    owner_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id    BIGINT REFERENCES repair_posts(id) ON DELETE CASCADE,
    listing_id BIGINT REFERENCES listings(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL CHECK (kind IN ('image', 'video')),
    path       TEXT NOT NULL,
    mime       TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE orders (
    id               BIGSERIAL PRIMARY KEY,
    customer_id      BIGINT NOT NULL REFERENCES users(id),
    seller_id        BIGINT NOT NULL REFERENCES users(id),
    listing_id       BIGINT NOT NULL REFERENCES listings(id),
    quantity         INT NOT NULL CHECK (quantity > 0),
    unit_price_tetri BIGINT NOT NULL,
    delivery         BOOLEAN NOT NULL,
    delivery_address TEXT NOT NULL DEFAULT '',
    delivery_fee_tetri BIGINT NOT NULL DEFAULT 0,
    total_tetri      BIGINT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'confirmed', 'out_for_delivery', 'completed', 'cancelled')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE wallet_transactions (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_tetri BIGINT NOT NULL,
    reason       TEXT NOT NULL,
    reference    TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE boosts (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_tetri BIGINT NOT NULL CHECK (amount_tetri > 0),
    starts_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    ends_at      TIMESTAMPTZ NOT NULL
);
CREATE INDEX boosts_active_idx ON boosts (user_id, ends_at);
