# eliava

Marketplace for Georgia with three services:

- **Technicians** list their specialities and location; customers find them by speciality and distance.
- **Customers** post a repair request (description, photos/videos, location) and get a price estimate. Nearby technicians see the request and pay to view the customer's number. Every view is logged: the customer and the admin can see which technician viewed it.
- **Market**: sellers list parts and appliances by category with price and stock, and pick their Eliava market stall (A5, B2, ...) on a map. Buyers order with pickup or home delivery.

Also included: paid registration for technicians and sellers, paid leaderboard boosts (the highest active boost ranks first and is shown first in search), duplicate-post flagging, an admin page, and draft Terms & Conditions (`web/terms.html`).

## Structure

- `backend/` — Go API (`backend/cmd/api`, code in `backend/internal/server`, SQL migrations in `backend/internal/server/migrations`, applied automatically on start)
- `web/` — the website (plain HTML/JS, Leaflet + OpenStreetMap maps). The Go server serves it, or host it separately (e.g. Vercel) and set `window.ELIAVA_API` in `web/config.js`.
- `frontend` — an older submodule pointer with no `.gitmodules`; its files aren't in this repo.

## Run locally

With Docker: `docker compose up --build`, then open http://localhost:8080. Register with `admin@example.com` to get the admin account.

Without Docker: install PostgreSQL with the PostGIS extension, create a database `eliava`, then:

```powershell
cd backend
$env:DB_PASSWORD = "your-postgres-password"
$env:ADMIN_EMAIL = "you@example.com"
go run ./cmd/api
```

Open http://localhost:8080.

## Configuration (environment variables)

| Variable | Default | Meaning |
| --- | --- | --- |
| `DATABASE_URL` | — | Postgres URL (Supabase/Neon). If empty, `DB_HOST`/`DB_PORT`/`DB_USER`/`DB_NAME`/`DB_PASSWORD` are used |
| `PORT` | `8080` | HTTP port |
| `ADMIN_EMAIL` | — | The account with this email becomes admin |
| `ALLOWED_ORIGINS` | `http://localhost:3000,http://localhost:8080` | Sites allowed to call the API (add your Vercel URL) |
| `UPLOAD_DIR` | `uploads` | Where photos/videos are stored |
| `SUPABASE_URL`, `SUPABASE_SERVICE_KEY`, `SUPABASE_BUCKET` | —, —, `media` | If set, photos/videos go to this public Supabase Storage bucket (created automatically) instead of `UPLOAD_DIR`. Needed on hosts that wipe the disk, like Render free |
| `PAYMENTS_MODE` | `mock` | `mock` = test top-ups with fake money. Anything else disables top-ups until a real provider is connected |
| `REGISTRATION_FEE_TETRI` / `REGISTRATION_DAYS` | `2000` / `30` | Technician/seller registration: 20 GEL per 30 days |
| `REVEAL_PRICE_TETRI` | `200` | Price for a technician to view a customer's number (2 GEL) |
| `DELIVERY_FEE_TETRI` | `500` | Delivery fee (5 GEL) |
| `BOOST_DAYS` | `30` | How long a leaderboard boost lasts |
| `AI_API_KEY`, `AI_BASE_URL`, `AI_MODEL` | —, OpenAI, `gpt-4o-mini` | Optional. With a key, AI refines the price estimate and decides duplicates. Without one, built-in rules are used |

Money is stored in tetri (1 GEL = 100 tetri).

## Tests

```bash
docker run -d --name eliava-db -e POSTGRES_PASSWORD=devpass -p 5432:5432 postgis/postgis:16-3.4
docker exec eliava-db psql -U postgres -c "CREATE DATABASE eliava_test"
cd backend && TEST_DATABASE_URL=postgres://postgres:devpass@localhost:5432/eliava_test go test ./...
```

## Not done yet

- Real payments: top-ups are simulated. Connect a Georgian provider (Bank of Georgia iPay, TBC Pay or Flitt) in `backend/internal/server/wallet.go` (`topUp`).
- Phone number verification by SMS.
- Delivery is a flat fee handled by the seller/courier manually.
- Terms & Conditions are a draft and need review by a Georgian lawyer.
