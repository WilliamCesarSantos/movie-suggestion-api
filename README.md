# Movie Suggestion App

A production-ready movie recommendation API built in Go, using clean architecture principles and a graph database (Neo4j) to power multiple recommendation algorithms.

## Overview

The Movie Suggestion App provides personalized movie recommendations to users by automatically selecting the best algorithm based on their watch history. It integrates with OMDB for movie data, stores auth users in PostgreSQL, handles JWT auth natively in Go, and processes async import jobs via SQS.

---

## Architecture

The project follows **Clean Architecture** with three layers:

- **Domain** (`internal/domain/`) — Entities, repository interfaces, use case interfaces. Zero external dependencies.
- **Application** (`internal/application/`) — Use case implementations, algorithm selection and dispatching logic.
- **Infrastructure** (`internal/infrastructure/`) — Neo4j/PostgreSQL repositories, auth services, HTTP handlers/middleware/router, AWS SQS client, OMDB client, observability (metrics + tracing).

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Client / API Consumer                      │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ HTTP
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          Go API (chi router)                        │
│  ┌──────────────┐  ┌───────────────┐  ┌─────────────────────────┐  │
│  │ JWT/RBAC     │  │  Observability│  │  HTTP Handlers          │  │
│  │ Middleware   │  │  (OTEL/Prom)  │  │  (User/Movie/Auth)      │  │
│  └──────┬───────┘  └───────────────┘  └────────────┬────────────┘  │
│         │                                           │               │
│  ┌──────▼───────────────────────────────────────────▼────────────┐  │
│  │                    Application Use Cases                       │  │
│  │ SuggestMovies | ManageUser | Login | ImportMovies | GetMovie   │  │
│  └──────────────────────────────┬────────────────────────────────┘  │
│                                 │                                    │
│  ┌──────────────────────────────▼─────────────────────────────────┐ │
│  │                  Algorithm Layer                               │ │
│  │  Popular | ContentBased | Collaborative | Hybrid | Serendipity│ │
│  └──────────────────────────────┬────────────────────────────────┘ │
└─────────────────────────────────┼───────────────────────────────────┘
                                  │
         ┌────────────────────────┼────────────────────────────┐
         │                        │                            │
         ▼                        ▼                            ▼
 ┌─────────────┐         ┌──────────────┐              ┌──────────────┐
 │   Neo4j     │         │ PostgreSQL   │              │  AWS SQS     │
 │  (Graph DB) │         │  (auth data) │              │  (import q)  │
 └─────────────┘         └──────────────┘              └──────┬───────┘
                                                               │
                                                      ┌────────▼───────┐
                                                      │  SQS Consumer  │
                                                      │  (Go workers)  │
                                                      └────────┬───────┘
                                                               │
                                                      ┌────────▼───────┐
                                                      │    OMDB API    │
                                                      └────────────────┘
```

### Recommendation Algorithm Auto-Selection

| Watch Count   | Algorithm Selected |
|---------------|--------------------|
| < 5 watched   | POPULAR            |
| 5–19 watched  | CONTENT_BASED      |
| ≥ 20 watched  | COLLABORATIVE      |

Users can also override the algorithm via query parameter.

---

## Prerequisites

- [Go 1.22+](https://golang.org/dl/)
- [Docker](https://www.docker.com/)
- [Docker Compose](https://docs.docker.com/compose/)

---

## Quick Start

### With Docker Compose (full local environment)

```bash
docker-compose up -d
```

This starts:
- **Neo4j** (port 7474/7687)
- **LocalStack** (SQS, Lambda, SSM — port 4566)
- **Jaeger** (tracing UI — port 16686, OTLP — port 4317)
- **Prometheus** (port 9091)
- **Grafana** (port 3000)
- **API** (port 8080, metrics port 9090)

### Run API locally (without Docker)

```bash
# Start only the infrastructure
docker-compose up -d neo4j localstack jaeger

# Set environment variables
export NEO4J_URI=bolt://localhost:7687
export NEO4J_USERNAME=neo4j
export NEO4J_PASSWORD=password
export AWS_REGION=us-east-1
export AWS_ENDPOINT=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test

# Run the API
go run ./cmd/api
```

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `NEO4J_URI` | `bolt://localhost:7687` | Neo4j connection URI |
| `NEO4J_USERNAME` | `neo4j` | Neo4j username |
| `NEO4J_PASSWORD` | `password` | Neo4j password |
| `NEO4J_DATABASE` | `neo4j` | Neo4j database name |
| `OMDB_BASE_URL` | `http://www.omdbapi.com` | OMDB API base URL |
| `OMDB_API_KEY` | _(empty)_ | OMDB API key |
| `OMDB_TIMEOUT_SECONDS` | `10` | OMDB HTTP client timeout |
| `RECOMMENDATION_DEFAULT_LIMIT` | `10` | Default number of recommendations |
| `RECOMMENDATION_MAX_LIMIT` | `50` | Maximum allowed recommendations |
| `RECOMMENDATION_HYBRID_CONTENT_WEIGHT` | `0.5` | Hybrid: content-based weight |
| `RECOMMENDATION_HYBRID_COLLABORATIVE_WEIGHT` | `0.5` | Hybrid: collaborative weight |
| `RECOMMENDATION_MIN_IMDB_RATING` | `6.0` | Minimum IMDb rating filter |
| `RECOMMENDATION_SERENDIPITY_MIN_RATING` | `5.0` | Serendipity minimum rating |
| `RECOMMENDATION_CONTENT_BASED_MIN_WATCHES` | `5` | Watches needed for content-based |
| `RECOMMENDATION_COLLABORATIVE_MIN_WATCHES` | `20` | Watches needed for collaborative |
| `RECOMMENDATION_CONTENT_PREFERENCE_THRESHOLD` | `0.7` | Content preference threshold |
| `SQS_QUEUE_URL` | `http://localhost:4566/000000000000/movie-import` | SQS queue URL |
| `SQS_WORKER_COUNT` | `5` | Number of SQS consumer workers |
| `AWS_REGION` | `us-east-1` | AWS region |
| `AWS_ENDPOINT` | _(empty)_ | Custom AWS endpoint (for LocalStack) |
| `POSTGRES_DSN` | `postgres://postgres:password@localhost:5432/movie_suggestion?sslmode=disable` | PostgreSQL DSN for auth data |
| `ARGON2_PEPPER` | `movie-suggestion-123456` | Pepper used before Argon2id hashing |
| `JWT_SECRET` | `dev-secret` | JWT signing secret |
| `JWT_EXPIRY_HOURS` | `24` | JWT expiration in hours |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OpenTelemetry OTLP endpoint |
| `OTEL_SERVICE_NAME` | `movie-suggestion` | Service name for traces |
| `SERVER_PORT` | `8080` | HTTP API port |
| `METRICS_PORT` | `9090` | Prometheus metrics port |
| `LOG_PRETTY` | `false` | Enable pretty console logging |

---

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v1/health` | None | Health check |
| `POST` | `/api/v1/login` | None | Login and get a JWT |
| `POST` | `/api/v1/users` | Bearer (`users:write`) | Create a new user |
| `GET` | `/api/v1/users/{id}` | Bearer (`users:read` + owner or `*`) | Get user details |
| `GET` | `/api/v1/movies` | Bearer (`movies:read` or `movies:write`) | Get personalized movie recommendations |
| `GET` | `/api/v1/movies/{id}` | Bearer (`movies:read`) | Get full movie details |
| `POST` | `/api/v1/movies/{id}/watched` | Bearer (`movies-watch:write`) | Record a watched movie for the authenticated user |
| `POST` | `/api/v1/movie-import` | Bearer (`movies:write`) | Trigger movie import |
| `GET` | `/metrics` | None (port 9090) | Prometheus metrics |

### Query Parameters for Recommended Movies

| Parameter | Type | Description |
|-----------|------|-------------|
| `limit` | int | Number of recommendations per page (default: 10, max: 50) |
| `cursor` | string | Opaque cursor token for pagination |
| `algorithm` | string | Override algorithm: `POPULAR`, `CONTENT_BASED`, `COLLABORATIVE`, `HYBRID`, `SERENDIPITY` |
| `title` | string | Case-insensitive partial title filter applied together with algorithm results |

> User ID is extracted automatically from the Bearer token — no path parameter required.

---

## Curl Examples

### Health Check

```bash
curl http://localhost:8080/api/v1/health
```

### Login

```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"email": "william_cesar_santos@hotmail.com", "password": "123456"}'
```

### Create a User

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com","password":"s3cr3t","roles":["users:read","users:write","movies:read","movie-watch:write"]}'
```

### Get User Details

```bash
curl http://localhost:8080/api/v1/users/<user-id> \
  -H "Authorization: Bearer <token>"
```

### Record a Watched Movie

```bash
curl -X POST http://localhost:8080/api/v1/movies/<movie-id>/watched \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"rating": 8.5, "reaction": "liked"}'
```

### Get Recommended Movies

```bash
# Auto-selected algorithm
curl "http://localhost:8080/api/v1/movies?limit=10" \
  -H "Authorization: Bearer <token>"

# Override algorithm
curl "http://localhost:8080/api/v1/movies?limit=5&algorithm=SERENDIPITY" \
  -H "Authorization: Bearer <token>"

# Refine by title (contains, case-insensitive)
curl "http://localhost:8080/api/v1/movies?limit=5&title=matrix" \
  -H "Authorization: Bearer <token>"
```

### Get Movie Details

```bash
curl http://localhost:8080/api/v1/movies/<movie-id> \
  -H "Authorization: Bearer <token>"
```

> **`GET /api/v1/movies`** returns personalized recommendations with pagination fields: `data`, `nextCursor`, `prevCursor`, `hasNext`, `hasPrev`, `limit`, `count`, `total`.
>
> **`GET /api/v1/movies/{id}`** returns the full movie object, including all available fields: `id`, `title`, `year`, `plot`, `runtime`, `poster`, `imdbRating`, `imdbId`, `genres`, `actors`, `directors`.

### Trigger Movie Import

```bash
curl -X POST http://localhost:8080/api/v1/movie-import \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"searchTerms": ["inception", "matrix", "interstellar", "tropa", "compadecida", "chefão" "pulp fiction"], "maxPages": 3}'
```

---

## Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/application/recommendation/...
go test ./internal/application/usecase/...
```

---

## Project Structure

```
movie-suggestion/
├── cmd/
│   ├── api/
│   │   └── main.go                    # Application entrypoint
│   └── seed/
│       └── main.go                    # Argon2id hash helper
├── config/
│   └── config.go                      # Configuration loading from env vars
├── internal/
│   ├── domain/
│   │   ├── entity/                    # Domain entities (Movie, User, AuthUser, errors)
│   │   ├── repository/                # Repository interfaces
│   │   └── usecase/                   # Use case interfaces
│   ├── application/
│   │   ├── recommendation/                # Algorithm selector and dispatcher
│   │   └── usecase/                   # Use case implementations
│   └── infrastructure/
│       ├── auth/                      # JWT and password services
│       ├── http/
│       │   ├── handler/               # HTTP handlers
│       │   ├── middleware/            # Auth, RBAC, observability middleware
│       │   └── router/                # Chi router setup
│       ├── neo4j/
│       │   ├── cypher/                # Cypher query constants
│       │   ├── movie_repository.go
│       │   ├── user_repository.go
│       │   └── recommendation_repository.go
│       ├── postgres/                  # PostgreSQL auth repository
│       ├── observability/             # Prometheus metrics, OTEL tracer
│       ├── omdb/                      # OMDB HTTP client and adapter
│       └── sqs/                       # SQS consumer and publisher
├── nginx/
│   └── nginx.conf
├── scripts/
│   ├── aws/                           # LocalStack initialization scripts
│   ├── postgresql/                    # PostgreSQL bootstrap scripts
│   ├── prometheus/                    # Prometheus configuration
│   ├── grafana/                       # Grafana provisioning
│   └── demo.sh                        # End-to-end demo script
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── go.sum
```
