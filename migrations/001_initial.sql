-- migrations/001_initial.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- -------------------------------------------------------
-- invoices
-- -------------------------------------------------------
CREATE TABLE invoices (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reference_code  TEXT NOT NULL UNIQUE,
    expected_amount NUMERIC(14,2) NOT NULL,
    status          TEXT NOT NULL DEFAULT 'PENDING',
    customer_ref    TEXT,
    due_date        TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invoices_reference_code ON invoices(reference_code);
CREATE INDEX idx_invoices_status ON invoices(status);

-- -------------------------------------------------------
-- transactions
-- -------------------------------------------------------
CREATE TABLE transactions (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reference_id        UUID NOT NULL,                  -- our poll handle (X-Reference-Id sent to MoMo)
    external_id         TEXT NOT NULL,                  -- = invoice.reference_code
    momo_transaction_id TEXT UNIQUE,                    -- MTN financial TX ID (nullable until confirmed)
    amount              NUMERIC(14,2) NOT NULL,
    phone_number        TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'PENDING',
    last_polled_at      TIMESTAMPTZ,
    poll_count          INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_reference_id  ON transactions(reference_id);
CREATE INDEX idx_transactions_external_id   ON transactions(external_id);  -- no FK; enforced in service
CREATE INDEX idx_transactions_status        ON transactions(status);

-- -------------------------------------------------------
-- momo_events  (raw callback payloads — never mutate)
-- -------------------------------------------------------
CREATE TABLE momo_events (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    payload     JSONB NOT NULL,
    processed   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_momo_events_processed ON momo_events(processed) WHERE processed = FALSE;
