# Shopping List App

A Go-based REST API backend for a household shopping and meal planning application. It lets users manage a shared shopping list, plan weekly meals, track pantry stock, and coordinate across a shared household.

## Features

- **User accounts** — registration, login, and session-based authentication with bcrypt password hashing
- **Households** — create a household, invite others via a shareable code, and manage membership approvals
- **Shopping list** — add, remove, and categorise items; mark items as "have it" without removing them from the list
- **Meals & meal planning** — define meals with ingredients, set portion sizes, and link meals to a weekly plan so their ingredients populate the shopping list automatically
- **Pantry** — track perishable stock with expiry awareness; a background scheduler marks items as `expiring_soon` or `expired` every hour
- **Item catalogue** — a typed catalogue of shopping items covering 18 categories (fruit, dairy, meat, bakery, household goods, etc.)

## Tech Stack

- **Language:** Go
- **Database:** PostgreSQL (via [pgx v5](https://github.com/jackc/pgx))
- **Migrations:** [golang-migrate](https://github.com/golang-migrate/migrate)
- **Query generation:** [sqlc](https://sqlc.dev/)
- **Config:** [godotenv](https://github.com/joho/godotenv)

## Prerequisites

- Go 1.25+
- PostgreSQL instance
- A `.env` file in the project root (see [Configuration](#configuration))

## Getting Started

```bash
# Clone the repository
git clone <repo-url>
cd ShoppingListApp-main

# Install dependencies (vendored, so this is optional)
go mod download

# Run the server
go run ./cmd/api
```

The server starts on **`:8080`** and applies any pending database migrations automatically on startup.

## Configuration

Create a `.env` file in the project root. At minimum you will need a PostgreSQL connection string:

```env
DATABASE_URL=postgres://user:password@localhost:5432/shopping_db?sslmode=disable
```

## API Routes

Routes are grouped by feature area and protected by session authentication unless marked public.

| Area | Prefix | Notes |
|---|---|---|
| Authentication | `/` | `/register` and `/login` are public |
| Users | `/users` | Authenticated |
| Households | `/households` | Authenticated |
| Shopping list | `/shopping-list` | Authenticated |
| Meals | `/meals` | Authenticated |
| Pantry | `/pantry` | Authenticated |

## Database Migrations

Migrations run automatically on startup. If a migration gets stuck in a dirty state, you can force-reset it with:

```bash
go run ./cmd/api -force-migration=<version>
```

This clears the dirty state so the server can start normally on the next run.

## Running Tests

```bash
go test ./...
```

Tests are also run automatically on every push and pull request via GitHub Actions (`.github/workflows/test.yml`).

## Project Structure

```
cmd/api/          # Application entry point
authentication/   # Login, registration, session management
user/             # User profile logic
households/       # Household creation, invites, membership
shoppingList/     # Shopping list and "have it" tracking
meals/            # Meal definitions, ingredients, meal plan
pantry/           # Perishable stock tracking and expiry scheduler
database/
  migrations/     # SQL migration files
  queries/        # Raw SQL queries
  sqlc/           # Generated query code (sqlc)
  schemas/        # Full schema reference
internal/api/
  httpx/          # Generic router and endpoint helpers
  middleware/     # CORS and middleware chain
testing/          # Shared fakes and service-level integration tests
```
