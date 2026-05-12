# Movie Suggestion App

A production-ready movie recommendation API built in Go, using clean architecture principles and a graph database (Neo4j) to power multiple recommendation algorithms.

## Overview

The Movie Suggestion App provides personalized movie recommendations to users by automatically selecting the best algorithm based on their watch history. It integrates with OMDB for movie data, uses AWS Lambda for JWT auth and import triggering, and processes async import jobs via SQS.

---

## Architecture

The project follows **Clean Architecture** with three layers:

- **Domain** (`internal/domain/`) — Entities, repository interfaces, use case interfaces. Zero external dependencies.
- **Application** (`internal/application/`) — Use case implementations, algorithm selection and dispatching logic.
- **Infrastructure** (`internal/infrastructure/`) — Neo4j repositories, HTTP handlers/middleware/router, AWS clients (Lambda, SQS), OMDB client, observability (metrics + tracing).

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
│  │ Auth Lambda  │  │  Observability│  │  HTTP Handlers          │  │
│  │ Middleware   │  │  (OTEL/Prom)  │  │  (User/Movie/Admin)     │  │
│  └──────┬───────┘  └───────────────┘  └────────────┬────────────┘  │
│         │                                           │               │
│  ┌──────▼───────────────────────────────────────────▼────────────┐  │
│  │                    Application Use Cases                       │  │
│  │  SuggestMovies | ManageUser | ImportMovies | ProcessImport     │  │
│  └──────────────────────────────┬─────────────────────────────── ┘  │
│                                 │                                    │
│  ┌──────────────────────────────▼─────────────────────────────────┐ │
│  │                  Algorithm Layer                               │ │
│  │  Popular | ContentBased | Collaborative | Hybrid | Serendipity│ │
│  └──────────────────────────────┬─────────────────────────────── ┘ │
└─────────────────────────────────┼───────────────────────────────────┘
                                  │
        ┌─────────────────────────┼──────────────────────┐
        │                         │                      │
        ▼                         ▼                      ▼
 ┌─────────────┐          ┌──────────────┐      ┌──────────────┐
 │   Neo4j     │          │  AWS Lambda  │      │  AWS SQS     │
 │  (Graph DB) │          │  auth/import │      │  (import q)  │
 └─────────────┘          └──────────────┘      └──────┬───────┘
                                                        │
                                               ┌────────▼───────┐
                                               │  SQS Consumer  │
                                               │  (Go workers)  │
                                               └────────┬───────┘
                                                        │
                                               ┌────────▼───────┐
                                               │  OMDB API      │
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
| `SUGGESTION_DEFAULT_LIMIT` | `10` | Default number of suggestions |
| `SUGGESTION_MAX_LIMIT` | `50` | Maximum allowed suggestions |
| `SUGGESTION_HYBRID_CONTENT_WEIGHT` | `0.5` | Hybrid: content-based weight |
| `SUGGESTION_HYBRID_COLLABORATIVE_WEIGHT` | `0.5` | Hybrid: collaborative weight |
| `SUGGESTION_MIN_IMDB_RATING` | `6.0` | Minimum IMDb rating filter |
| `SUGGESTION_SERENDIPITY_MIN_RATING` | `5.0` | Serendipity minimum rating |
| `SUGGESTION_CONTENT_BASED_MIN_WATCHES` | `5` | Watches needed for content-based |
| `SUGGESTION_COLLABORATIVE_MIN_WATCHES` | `20` | Watches needed for collaborative |
| `SUGGESTION_CONTENT_PREFERENCE_THRESHOLD` | `0.7` | Content preference threshold |
| `SQS_QUEUE_URL` | `http://localhost:4566/000000000000/movie-import` | SQS queue URL |
| `SQS_WORKER_COUNT` | `5` | Number of SQS consumer workers |
| `AWS_REGION` | `us-east-1` | AWS region |
| `AWS_ENDPOINT` | _(empty)_ | Custom AWS endpoint (for LocalStack) |
| `LAMBDA_AUTH_FUNCTION_NAME` | `auth-function` | Auth Lambda function name |
| `LAMBDA_IMPORT_FUNCTION_NAME` | `import-function` | Import Lambda function name |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OpenTelemetry OTLP endpoint |
| `OTEL_SERVICE_NAME` | `movie-suggestion` | Service name for traces |
| `SERVER_PORT` | `8080` | HTTP API port |
| `METRICS_PORT` | `9090` | Prometheus metrics port |
| `LOG_PRETTY` | `false` | Enable pretty console logging |
| `JWT_SECRET` | `dev-secret` | JWT signing secret |

---

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v1/health` | None | Health check |
| `POST` | `/api/v1/users` | None | Create a new user |
| `GET` | `/api/v1/users/{id}` | Bearer (self or admin) | Get user details |
| `POST` | `/api/v1/users/{id}/watched` | Bearer (self or admin) | Record a watched movie |
| `POST` | `/api/v1/users/{id}/liked` | Bearer (self or admin) | Record a liked movie |
| `POST` | `/api/v1/users/{id}/disliked` | Bearer (self or admin) | Record a disliked movie |
| `GET` | `/api/v1/users/{id}/suggestions` | Bearer (self or admin) | Get movie suggestions |
| `GET` | `/api/v1/movies/{id}` | Bearer | Get movie details |
| `POST` | `/api/v1/admin/import/trigger` | Bearer (admin only) | Trigger movie import |
| `GET` | `/metrics` | None (port 9090) | Prometheus metrics |

### Query Parameters for Suggestions

| Parameter | Type | Description |
|-----------|------|-------------|
| `limit` | int | Number of suggestions (default: 10, max: 50) |
| `algorithm` | string | Override algorithm: `POPULAR`, `CONTENT_BASED`, `COLLABORATIVE`, `HYBRID`, `SERENDIPITY` |

---

## Curl Examples

### Health Check

```bash
curl http://localhost:8080/api/v1/health
```

### Create a User

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "email": "alice@example.com"}'
```

### Get a Token (via Auth Lambda — example with LocalStack)

```bash
aws --endpoint-url=http://localhost:4566 lambda invoke \
  --function-name auth-function \
  --payload '{"action":"generate","userId":"<user-id>","email":"alice@example.com","role":"user"}' \
  /dev/stdout
```

### Get User Details

```bash
curl http://localhost:8080/api/v1/users/<user-id> \
  -H "Authorization: Bearer <token>"
```

### Record a Watched Movie

```bash
curl -X POST http://localhost:8080/api/v1/users/<user-id>/watched \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"movieId": "<movie-id>", "rating": 8.5}'
```

### Record a Liked Movie

```bash
curl -X POST http://localhost:8080/api/v1/users/<user-id>/liked \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"movieId": "<movie-id>", "algorithm": "POPULAR"}'
```

### Record a Disliked Movie

```bash
curl -X POST http://localhost:8080/api/v1/users/<user-id>/disliked \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"movieId": "<movie-id>"}'
```

### Get Movie Suggestions

```bash
# Auto-selected algorithm
curl "http://localhost:8080/api/v1/users/<user-id>/suggestions?limit=10" \
  -H "Authorization: Bearer <token>"

# Override algorithm
curl "http://localhost:8080/api/v1/users/<user-id>/suggestions?limit=5&algorithm=SERENDIPITY" \
  -H "Authorization: Bearer <token>"
```

### Get Movie Details

```bash
curl http://localhost:8080/api/v1/movies/<movie-id> \
  -H "Authorization: Bearer <token>"
```

### Trigger Movie Import (Admin)

```bash
curl -X POST http://localhost:8080/api/v1/admin/import/trigger \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"searchTerms": ["inception", "matrix", "interstellar"], "maxPages": 3}'
```

---

## Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/application/suggestion/...
go test ./internal/application/usecase/...
```

---

## Project Structure

```
movie-suggestion/
├── cmd/
│   └── api/
│       └── main.go                    # Application entrypoint
├── config/
│   └── config.go                      # Configuration loading from env vars
├── internal/
│   ├── domain/
│   │   ├── entity/                    # Domain entities (Movie, User, errors)
│   │   ├── repository/                # Repository interfaces
│   │   └── usecase/                   # Use case interfaces
│   ├── application/
│   │   ├── suggestion/                # Algorithm selector, dispatcher, placeholders
│   │   └── usecase/                   # Use case implementations
│   └── infrastructure/
│       ├── http/
│       │   ├── handler/               # HTTP handlers
│       │   ├── middleware/            # Auth, observability middleware
│       │   └── router/                # Chi router setup
│       ├── lambda/                    # AWS Lambda clients (auth, import)
│       ├── neo4j/
│       │   ├── cypher/                # Cypher query constants
│       │   ├── movie_repository.go
│       │   ├── user_repository.go
│       │   └── suggestion_repository.go
│       ├── observability/             # Prometheus metrics, OTEL tracer
│       ├── omdb/                      # OMDB HTTP client
│       └── sqs/                       # SQS consumer
├── auth-lambda/                       # Python JWT auth Lambda
│   ├── handler.py
│   ├── jwt_service.py
│   ├── requirements.txt
│   └── Dockerfile
├── import-lambda/                     # Python import orchestration Lambda
│   ├── handler.py
│   ├── omdb_client.py
│   ├── sqs_publisher.py
│   ├── requirements.txt
│   └── Dockerfile
├── init/
│   └── neo4j-init.cypher              # Neo4j constraints and indexes
├── scripts/
│   ├── aws/                           # LocalStack initialization scripts
│   ├── prometheus/                    # Prometheus configuration
│   └── grafana/                       # Grafana provisioning
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── go.sum
```
