# momo-recon

MTN MoMo reconciliation system — Go backend.

## Prerequisites

- Go 1.22+
- PostgreSQL 14+
- MTN MoMo sandbox credentials (https://momodeveloper.mtn.com)
- ngrok or similar for local callback URL

## Setup

```bash
# 1. Clone and install deps
go mod tidy

# 2. Create database
createdb momo_recon

# 3. Run migration
psql momo_recon < migrations/001_create_tables.sql

# 4. Configure environment
cp .env.example .env
# Edit .env with your credentials

# 5. Expose localhost for MoMo callbacks (dev only)
ngrok http 8080
# Paste the ngrok URL into MOMO_CALLBACK_URL in .env

# 6. Run
go run ./cmd/server
```

## API

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/invoices | Create invoice |
| GET | /api/invoices | List invoices |
| GET | /api/invoices/{code} | Get invoice by reference code |
| POST | /api/pay | Initiate MoMo payment |
| GET | /api/transactions | List transactions |
| POST | /api/momo/callback | MoMo webhook (called by MTN) |
| GET | /health | Health check |

## Example flow

```bash
# 1. Create invoice
curl -X POST http://localhost:8080/api/invoices \
  -H "Content-Type: application/json" \
  -d '{"amount": 50, "customer_ref": "customer-123"}'
# -> {"reference_code": "INV-XXXXXXXX", ...}

# 2. Initiate payment
curl -X POST http://localhost:8080/api/pay \
  -H "Content-Type: application/json" \
  -d '{"reference_code": "INV-XXXXXXXX", "phone_number": "233XXXXXXXXX"}'
# -> {"status": "PENDING", ...}

# 3. Poll invoice status (or wait for callback to fire)
curl http://localhost:8080/api/invoices/INV-XXXXXXXX
# -> {"status": "PAID", ...}
```

## Architecture

```
Handler → Service → Repository → PostgreSQL
            ↓
         MoMo Client → MTN API
            ↓
         Event Worker  (processes callbacks async, every 5s)
         Polling Worker (polls pending txs, every 3min, max 10 polls)
```
