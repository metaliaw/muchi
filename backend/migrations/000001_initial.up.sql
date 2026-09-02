CREATE TABLE search_jobs (
    id text PRIMARY KEY,
    status text NOT NULL CHECK (status IN ('queued','running','completed','completed_with_errors','failed','cancelled')),
    total integer NOT NULL CHECK (total > 0), processed integer NOT NULL DEFAULT 0,
    found integer NOT NULL DEFAULT 0, not_found integer NOT NULL DEFAULT 0, errors integer NOT NULL DEFAULT 0,
    current_card text NOT NULL DEFAULT '', options jsonb NOT NULL DEFAULT '{}',
    incident_id text NOT NULL DEFAULT '', last_error text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL, started_at timestamptz, updated_at timestamptz NOT NULL, finished_at timestamptz
);
CREATE INDEX search_jobs_queue_idx ON search_jobs (status, created_at);

CREATE TABLE search_items (
    id text PRIMARY KEY, search_id text NOT NULL REFERENCES search_jobs(id) ON DELETE CASCADE,
    position integer NOT NULL, original_name text NOT NULL, normalized_name text NOT NULL,
    quantity integer NOT NULL CHECK (quantity BETWEEN 1 AND 99),
    verify_stock boolean NOT NULL DEFAULT true, stores_only boolean NOT NULL DEFAULT true,
    status text NOT NULL CHECK (status IN ('pending','running','found','not_found','source_error')),
    attempts integer NOT NULL DEFAULT 0, source text NOT NULL DEFAULT '', error_code text NOT NULL DEFAULT '', error_message text NOT NULL DEFAULT '',
    lease_owner text NOT NULL DEFAULT '', lease_until timestamptz, started_at timestamptz, updated_at timestamptz, finished_at timestamptz,
    UNIQUE (search_id, position)
);
CREATE INDEX search_items_claim_idx ON search_items (status, lease_until, search_id, position);

CREATE TABLE offers (
    id text PRIMARY KEY, search_item_id text NOT NULL REFERENCES search_items(id) ON DELETE CASCADE,
    card_name text NOT NULL, store text NOT NULL, price_amount numeric(20,2) NOT NULL CHECK (price_amount >= 0),
    price_currency char(3) NOT NULL, url text NOT NULL, variant_id text NOT NULL DEFAULT '', language text NOT NULL DEFAULT '',
    condition text NOT NULL DEFAULT '', finish text NOT NULL DEFAULT '', source text NOT NULL,
    stock_status text NOT NULL DEFAULT 'unknown', suspicious boolean NOT NULL DEFAULT false,
    suspicious_reason text NOT NULL DEFAULT '', metadata jsonb NOT NULL DEFAULT '{}', observed_at timestamptz NOT NULL DEFAULT NOW(),
    UNIQUE (search_item_id, store, url, variant_id)
);
CREATE INDEX offers_item_price_idx ON offers (search_item_id, price_amount);

CREATE TABLE source_health (
    source text PRIMARY KEY, last_success timestamptz, last_failure timestamptz,
    consecutive_failures integer NOT NULL DEFAULT 0, latency_ms bigint NOT NULL DEFAULT 0,
    circuit_open_until timestamptz, updated_at timestamptz NOT NULL DEFAULT NOW()
);
CREATE TABLE offer_cache (
    cache_key text PRIMARY KEY, kind text NOT NULL CHECK (kind IN ('positive','negative')),
    payload jsonb NOT NULL, expires_at timestamptz NOT NULL, updated_at timestamptz NOT NULL DEFAULT NOW()
);
CREATE INDEX offer_cache_expiry_idx ON offer_cache (expires_at);
CREATE TABLE traffic_leases (
    domain text PRIMARY KEY, next_allowed_at timestamptz NOT NULL, updated_at timestamptz NOT NULL
);
CREATE TABLE idempotency_keys (
    key text NOT NULL, action text NOT NULL, request_hash text NOT NULL, resource_id text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(), PRIMARY KEY (key, action)
);
