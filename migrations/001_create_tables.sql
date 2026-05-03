-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- invoices: represents an expected payment
CREATE TABLE IF NOT EXISTS invoices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_code  TEXT NOT NULL UNIQUE,
    expected_amount NUMERIC(12, 2) NOT NULL,
    status          TEXT NOT NULL DEFAULT 'PENDING',
    customer_ref    TEXT,
    due_date        TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invoices_reference_code ON invoices(reference_code);
CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(status);

-- transactions: represents a MoMo payment attempt/result
CREATE TABLE IF NOT EXISTS transactions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_id        UUID NOT NULL,                           -- sent to MoMo as the request UUID
    external_id         TEXT NOT NULL,                          -- = invoice.reference_code (no FK, enforced in service)
    momo_transaction_id TEXT UNIQUE,                            -- financialTransactionId; nullable until confirmed
    amount              NUMERIC(12, 2) NOT NULL,
    phone_number        TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'PENDING',
    last_polled_at      TIMESTAMPTZ,
    poll_count          INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transactions_external_id ON transactions(external_id);
CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status);
CREATE INDEX IF NOT EXISTS idx_transactions_reference_id ON transactions(reference_id);

-- momo_events: raw callback payloads — store first, process async
CREATE TABLE IF NOT EXISTS momo_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payload    JSONB NOT NULL,
    processed  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_momo_events_processed ON momo_events(processed) WHERE processed = FALSE;
