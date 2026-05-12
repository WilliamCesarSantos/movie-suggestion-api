# Movie Suggestion App — Prompt for Claude Code
---
## 1. Project Overview
**Movie Suggestion** is a REST API that recommends movies to users based on their watch history and interaction patterns (likes, dislikes). The system evolves its recommendation algorithm automatically as the user accumulates more interactions — starting with simple popularity-based suggestions and progressing toward collaborative filtering as the user's history grows.
### What problem does it solve?
Users often don't know what to watch next. This API learns from user behavior and selects the most appropriate recommendation strategy automatically, without requiring manual configuration.
### Key features
- **Adaptive recommendations**: the algorithm changes automatically based on how much the user has watched.
- **5 algorithms**: Popular, Content-Based, Collaborative, Hybrid, and Serendipity.
- **Movie import pipeline**: an admin triggers a Lambda function that searches OMDB, publishes to SQS, and a Go worker pool persists movies to Neo4j.
- **JWT authentication**: tokens are generated and validated by a Python Lambda running on LocalStack.
- **Full observability**: Prometheus metrics, OpenTelemetry tracing (Jaeger), structured JSON logging (zerolog), and a Grafana dashboard.
---
## 2. Technology Stack
| Layer | Technology | Reason |
|-------|-----------|--------|
| Main API | **Go 1.22** | Performance, simplicity, native concurrency for the SQS worker pool |
| Graph Database | **Neo4j 5** | Graph traversal natively models user→movie relationships; Cypher queries power all suggestion algorithms |
| Authentication | **Python 3.12 Lambda** (LocalStack) | Decoupled JWT service; easy to replace with AWS Lambda in production |
| Movie Import | **Python 3.12 Lambda** + **SQS** | Asynchronous import; decouples HTTP response (202) from actual data ingestion |
| External Movie Source | **OMDB API** | Free movie metadata API with title, genres, actors, directors, ratings |
| Observability | **Prometheus + Grafana + Jaeger** | Industry standard; metrics + distributed tracing + dashboards |
| Local AWS | **LocalStack 3** | Emulates Lambda, SQS, and SSM Parameter Store locally without AWS costs |
| Architecture | **Clean Architecture** | Separates domain logic from infrastructure; enables unit testing without external dependencies |
---
## 3. Architecture Overview
```
┌─────────────────────────────────────────────────────────────────────┐
│                          HTTP Client                                 │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ REST (port 8080)
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Go API  (internal/infrastructure/http)                              │
│  ┌──────────────┐  ┌─────────────────────────────────────────────┐  │
│  │  Middleware   │  │  Handlers (users, movies, suggestions,       │  │
│  │  auth.go      │  │           admin, health)                    │  │
│  │  observ.go    │  └──────────────────┬──────────────────────────┘  │
│  └──────┬────────┘                     │                             │
│         │ JWT validate                 │ calls use cases             │
│         ▼                              ▼                             │
│  ┌─────────────┐        ┌──────────────────────────────────────┐    │
│  │ Auth Lambda │        │  Application Use Cases               │    │
│  │  (LocalStack│        │  suggest_movies, manage_user,        │    │
│  │   / AWS)    │        │  import_movies, update_user_profile  │    │
│  └─────────────┘        └───────────────┬──────────────────────┘    │
└──────────────────────────────────────── │ ───────────────────────────┘
                                          │ repository interfaces
                                          ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Infrastructure                                                       │
│  ┌──────────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │ Neo4j Repository  │  │ OMDB Client  │  │ Import Lambda Client   │ │
│  │ (movies, users,   │  │ (fetch movie │  │ (async invoke)         │ │
│  │  suggestions)     │  │  metadata)   │  └─────────┬──────────────┘ │
│  └──────────────────┘  └──────────────┘            │                │
│            │                                        │ InvocationType │
│            │ Bolt                                   │    = Event     │
│            ▼                                        ▼                │
│       ┌─────────┐                         ┌────────────────────┐    │
│       │  Neo4j  │                         │  Import Lambda     │    │
│       │  Graph  │                         │  (Python, OMDB →   │    │
│       │   DB    │◄────────────────────────│   SQS publish)     │    │
│       └─────────┘   SQS Consumer          └────────────────────┘    │
│                      (worker pool                                    │
│                       5 goroutines)                                  │
└─────────────────────────────────────────────────────────────────────┘
```
### Clean Architecture layers
```
domain/          → Entities (Movie, User), repository interfaces, use case interfaces.
                   No imports from application or infrastructure.
application/     → Use case implementations, AlgorithmSelector (pure domain service).
                   Depends only on domain interfaces.
infrastructure/  → Neo4j repos, OMDB client, SQS consumer, Lambda clients, HTTP handlers.
                   Implements domain interfaces. Contains all external I/O.
config/          → Reads environment variables. Wires everything together.
cmd/api/main.go  → Manual dependency injection. Starts HTTP server + SQS worker pool.
```
---
## 4. Core Data Flow
### 4.1 Movie Import (Admin triggers)
```
Admin POST /api/v1/admin/import/trigger
  → Go API calls Import Lambda (async, InvocationType=Event)
  → returns 202 Accepted immediately
Import Lambda:
  → OMDB search by searchTerms (paginated)
  → publish {imdbId, title} to SQS queue for each movie found
SQS Consumer (Go, 5 goroutines):
  → receive message
  → check if imdbId already in Neo4j (skip if yes — idempotency #1)
  → fetch full movie details from OMDB (?i=imdbId&plot=full)
  → MERGE movie + relationships into Neo4j (idempotency #2)
  → delete SQS message
  → on error: do NOT delete → SQS retries → DLQ after 3 attempts
```
### 4.2 User Suggestion Request
```
User GET /api/v1/users/{id}/suggestions?limit=10
  → auth middleware: extract Bearer token → validate via Auth Lambda
  → authorization check: role=admin OR token.sub == {id}
  → SuggestMoviesUseCase:
      → load user profile (currentAlgorithm, watchCount, likeCount)
      → if ?algorithm override → use that algorithm
      → else → use user.currentAlgorithm
      → run Cypher query for selected algorithm
      → return ranked list of movies
```
### 4.3 User Interaction (watched / liked / disliked)
```
User POST /api/v1/users/{id}/watched  {movieId, rating}
  → persist (:User)-[:WATCHED {watchedAt, rating}]->(:Movie) in Neo4j
  → call AlgorithmSelector.Select(user) → determine new algorithm
  → update user.currentAlgorithm in Neo4j
  → return updated user profile
```
---
## 5. Algorithm Selector — Decision Logic
Located at `internal/application/suggestion/selector.go`.
**Pure domain service** — no database or HTTP calls. Receives a `User` struct and returns a `SuggestionAlgorithm`.
```
Rule 1: watchCount < 5                                        → POPULAR
Rule 2: watchCount >= 5 && watchCount < 20                   → CONTENT_BASED
Rule 3: watchCount >= 20                                     → COLLABORATIVE
Rule 4: >= 70% of likes share same genre/director/actor      → CONTENT_BASED (overrides rule 3)
Rule 5: liked a movie suggested by POPULAR                   → keep POPULAR (boost that genre weight)
Rule 6: liked a movie suggested by COLLABORATIVE             → migrate to COLLABORATIVE
Rule 7: ?algorithm=SERENDIPITY in query string               → SERENDIPITY (user-forced override)
```
The selected algorithm is always persisted back to `(:User {currentAlgorithm})` after every interaction.
---
## 6. Implementation Guide (step-by-step)
Follow this order to avoid dependency issues. Each step builds on the previous.
### Step 1 — Project Scaffolding
1. Create `go.mod` with module name `github.com/yourorg/movie-suggestion`
2. Create the full directory tree shown in Section 7
3. Add main dependencies:
   ```
   go get github.com/neo4j/neo4j-go-driver/v5
   go get github.com/go-chi/chi/v5
   go get github.com/rs/zerolog
   go get github.com/prometheus/client_golang
   go get go.opentelemetry.io/otel
   go get github.com/aws/aws-sdk-go-v2
   ```
### Step 2 — Domain Layer (`internal/domain/`)
1. **`entity/movie.go`** — `Movie` struct with all fields; `Genre`, `Actor`, `Director` value objects
2. **`entity/user.go`** — `User` struct; `SuggestionAlgorithm` enum with constants
3. **`entity/suggestion.go`** — Typed domain errors: `ErrMovieNotFound`, `ErrUserNotFound`, `ErrUnauthorized`, `ErrForbidden`
4. **`repository/movie_repository.go`** — `MovieRepository` interface: `FindByID`, `FindByImdbID`, `Upsert`
5. **`repository/user_repository.go`** — `UserRepository` interface: `Create`, `FindByID`, `UpdateProfile`, `RecordWatched`, `RecordLiked`, `RecordDisliked`
6. **`repository/suggestion_repository.go`** — `SuggestionRepository` interface: one method per algorithm
7. **`usecase/`** — interfaces for `SuggestMovies`, `ImportMovies`, `ManageUser`, `UpdateUserProfile`
### Step 3 — Application Layer (`internal/application/`)
1. **`suggestion/selector.go`** — `AlgorithmSelector` struct with `Select(user User) SuggestionAlgorithm`; implement the 7 rules; **no imports from infrastructure**
2. **`suggestion/popular.go`, `content_based.go`, `collaborative.go`, `hybrid.go`, `serendipity.go`** — each calls its respective `SuggestionRepository` method
3. **`suggestion/algorithm.go`** — dispatcher: receives algorithm enum → delegates to the right strategy
4. **`usecase/suggest_movies_impl.go`** — calls AlgorithmSelector + SuggestionRepository
5. **`usecase/manage_user_impl.go`** — create/get user, call AlgorithmSelector after each interaction
6. **`usecase/update_user_profile_impl.go`** — update algorithm after watch/like/dislike
7. **`usecase/import_movies_impl.go`** — calls ImportLambdaClient (just delegates; returns 202)
8. **`usecase/process_movie_import.go`** — SQS worker logic: idempotency check → OMDB fetch → Neo4j upsert
### Step 4 — Infrastructure: Neo4j (`internal/infrastructure/neo4j/`)
1. Create `cypher/` sub-package with one `.go` file per algorithm holding the Cypher query strings (see Section 11)
2. **`movie_repository.go`** — implement `MovieRepository` using the Neo4j Go driver; use `MERGE` for upserts
3. **`user_repository.go`** — implement `UserRepository`; `RecordWatched` creates `:WATCHED` rel; `RecordLiked` creates `:LIKED` rel; also updates `INTERESTED_IN` genre relationships
4. **`suggestion_repository.go`** — implement `SuggestionRepository`; map Cypher results to `Movie` entities
### Step 5 — Infrastructure: External Services
1. **`omdb/client.go`** — HTTP client with `Search(term, page)` and `FetchByImdbID(id)` methods; respect rate limits
2. **`sqs/consumer.go`** — goroutine pool (`IMPORT_WORKER_COUNT`); long polling loop; calls `ProcessMovieImportUseCase`; graceful shutdown via context
3. **`sqs/message.go`** — struct for SQS message body (`imdbId`, `title`, `requestedAt`)
4. **`lambda/auth_client.go`** — AWS SDK Lambda invoke with `action: validate/generate`
5. **`lambda/import_client.go`** — AWS SDK Lambda invoke with `InvocationType: Event` (async)
### Step 6 — Infrastructure: HTTP
1. **`middleware/auth.go`** — extract Bearer token → validate via `AuthLambdaClient` → inject `userId`+`role` into context → `401`/`403` guards
2. **`middleware/observability.go`** — inject `correlationId` → start OTEL span → record Prometheus `http_requests_total` and `http_request_duration_seconds`
3. **`handler/`** — one file per resource group: `user_handler.go`, `movie_handler.go`, `admin_handler.go`, `health_handler.go`
4. **`router/`** — wire routes to handlers; apply middleware chain
### Step 7 — Infrastructure: Observability
1. **`observability/metrics.go`** — register all Prometheus metrics (see Section 16); expose handler on `METRICS_PORT`
2. **`observability/tracer.go`** — initialize OTEL SDK → OTLP gRPC exporter → Jaeger; provide helper to extract/inject trace context
### Step 8 — Configuration (`config/config.go`)
1. Read all environment variables listed in Section 17
2. Validate required vars at startup; fail fast with clear error messages
3. Return a single `Config` struct passed to `main.go`
### Step 9 — Wiring (`cmd/api/main.go`)
1. Load `Config`
2. Initialize Neo4j driver → repositories
3. Initialize OMDB client
4. Initialize AWS SDK clients → Lambda clients, SQS client
5. Build use cases (inject repositories)
6. Build HTTP handlers (inject use cases)
7. Build router with middleware
8. Start SQS consumer in background goroutines
9. Start HTTP server on `SERVER_PORT`
10. Start metrics server on `METRICS_PORT`
11. Graceful shutdown: listen for `SIGTERM`/`SIGINT` → 30s timeout → stop SQS workers, drain HTTP connections, close Neo4j
### Step 10 — Python Lambdas (`auth-lambda/`, `import-lambda/`)
Implement exactly as shown in Sections 13 and 14. These run independently inside LocalStack.
### Step 11 — Infrastructure Files
1. **`init/neo4j-init.cypher`** — constraints and indexes (Section 10)
2. **`scripts/aws/localstack-init.sh`** — creates SQS queues + DLQ + deploys both Lambdas (Section 18)
3. **`docker-compose.yml`** — all services (Section 19)
4. **`Dockerfile`** — multi-stage Go build (Section 21)
### Step 12 — Tests
1. **`AlgorithmSelector` unit tests** — test all 7 rules in isolation; no mocks needed (pure function)
2. **Use case unit tests** — generate mocks with `mockgen` for all repository interfaces; test happy paths + error paths
3. All test files live alongside their implementation in `_test.go` files
---
## 7. Project Structure (Clean Architecture)
```
cmd/
  api/
    main.go
internal/
  domain/
    entity/
      movie.go
      user.go
      suggestion.go
    repository/
      movie_repository.go
      user_repository.go
      suggestion_repository.go
    usecase/
      suggest_movies.go
      import_movies.go
      manage_user.go
      update_user_profile.go
  application/
    usecase/
      suggest_movies_impl.go
      import_movies_impl.go
      manage_user_impl.go
      update_user_profile_impl.go
      process_movie_import.go
    suggestion/
      algorithm.go
      popular.go
      content_based.go
      collaborative.go
      hybrid.go
      serendipity.go
      selector.go
  infrastructure/
    neo4j/
      movie_repository.go
      user_repository.go
      suggestion_repository.go
      cypher/
    omdb/
      client.go
    sqs/
      consumer.go
      message.go
    lambda/
      auth_client.go
      import_client.go
    http/
      handler/
      middleware/
        auth.go
        observability.go
      router/
    observability/
      tracer.go
      metrics.go
config/
  config.go
auth-lambda/
  handler.py
  jwt_service.py
  requirements.txt
  Dockerfile
import-lambda/
  handler.py
  omdb_client.py
  sqs_publisher.py
  requirements.txt
  Dockerfile
init/
  neo4j-init.cypher
scripts/
  aws/
    localstack-init.sh
    parameter-store.sh
  grafana/
    dashboards/
      movie-suggestion-api.json
    provisioning/
      dashboards/dashboards.yml
      datasources/prometheus.yml
  prometheus/
    prometheus.yml
docker-compose.yml
Dockerfile
```
---
## 8. Domain Entities
### Movie
`id, title, year, plot, runtime, poster, imdbRating, imdbId, genres[], actors[], directors[]`
### User
`id, name, email, createdAt, currentAlgorithm, watchCount, likeCount, dislikeCount`
### SuggestionAlgorithm (enum)
`POPULAR | CONTENT_BASED | COLLABORATIVE | HYBRID | SERENDIPITY`
---
## 9. Neo4j Graph Model
**Nodes:**
```
(:Movie {id, title, year, plot, runtime, poster, imdbRating, imdbId})
(:Genre {name})
(:Actor {name, imdbId})
(:Director {name, imdbId})
(:User {id, name, email, createdAt, currentAlgorithm, watchCount, likeCount, dislikeCount})
```
**Relationships:**
```
(:Movie)-[:HAS_GENRE]->(:Genre)
(:Movie)-[:HAS_ACTOR {order}]->(:Actor)
(:Movie)-[:DIRECTED_BY]->(:Director)
(:User)-[:WATCHED {watchedAt, rating}]->(:Movie)
(:User)-[:LIKED]->(:Movie)
(:User)-[:DISLIKED]->(:Movie)
(:User)-[:INTERESTED_IN]->(:Genre)
```
> **Why Neo4j?** Suggestion algorithms are fundamentally graph traversals. Cypher expresses "find users who watched the same movies as me and liked things I haven't seen" in 4 lines of code. A relational database would need complex self-joins with much worse performance at scale.
---
## 10. init/neo4j-init.cypher
```cypher
CREATE CONSTRAINT movie_id IF NOT EXISTS FOR (m:Movie) REQUIRE m.imdbId IS UNIQUE;
CREATE CONSTRAINT user_id IF NOT EXISTS FOR (u:User) REQUIRE u.id IS UNIQUE;
CREATE CONSTRAINT user_email IF NOT EXISTS FOR (u:User) REQUIRE u.email IS UNIQUE;
CREATE CONSTRAINT genre_name IF NOT EXISTS FOR (g:Genre) REQUIRE g.name IS UNIQUE;
CREATE CONSTRAINT actor_imdb IF NOT EXISTS FOR (a:Actor) REQUIRE a.imdbId IS UNIQUE;
CREATE CONSTRAINT director_imdb IF NOT EXISTS FOR (d:Director) REQUIRE d.imdbId IS UNIQUE;
CREATE INDEX movie_title IF NOT EXISTS FOR (m:Movie) ON (m.title);
CREATE INDEX movie_rating IF NOT EXISTS FOR (m:Movie) ON (m.imdbRating);
CREATE INDEX movie_year IF NOT EXISTS FOR (m:Movie) ON (m.year);
CREATE INDEX user_algorithm IF NOT EXISTS FOR (u:User) ON (u.currentAlgorithm);
```
> Run this file once when Neo4j starts. The `docker-compose.yml` mounts it to `/var/lib/neo4j/import/init.cypher`. Execute manually via `cypher-shell` or via APOC on first boot.
---
## 11. Suggestion Algorithms — Cypher Queries
Each algorithm lives in `internal/infrastructure/neo4j/cypher/` and is called by `SuggestionRepository`.
### POPULAR
Recommends highly-rated movies in genres the user has already expressed interest in.
```cypher
MATCH (m:Movie)-[:HAS_GENRE]->(g:Genre)<-[:INTERESTED_IN]-(u:User {id: $userId})
WHERE NOT (u)-[:WATCHED]->(m)
  AND m.imdbRating >= $minRating
RETURN m ORDER BY m.imdbRating DESC, m.watchCount DESC
LIMIT $limit
```
### CONTENT_BASED
Recommends movies that share genres, actors, or directors with movies the user has already liked.
```cypher
MATCH (u:User {id: $userId})-[:LIKED]->(liked:Movie)
MATCH (liked)-[:HAS_GENRE|HAS_ACTOR|DIRECTED_BY]->(shared)<-[:HAS_GENRE|HAS_ACTOR|DIRECTED_BY]-(candidate:Movie)
WHERE NOT (u)-[:WATCHED]->(candidate)
  AND candidate.imdbRating >= $minRating
WITH candidate, COUNT(shared) AS score
RETURN candidate ORDER BY score DESC
LIMIT $limit
```
### COLLABORATIVE
Finds users with similar taste (watched the same movies) and recommends what they liked.
```cypher
MATCH (u:User {id: $userId})-[:WATCHED]->(m:Movie)<-[:WATCHED]-(similar:User)
WHERE similar.id <> $userId
WITH similar, COUNT(m) AS overlap ORDER BY overlap DESC LIMIT 20
MATCH (similar)-[:LIKED]->(candidate:Movie)
WHERE NOT (u)-[:WATCHED]->(candidate)
  AND candidate.imdbRating >= $minRating
WITH candidate, COUNT(similar) AS score
RETURN candidate ORDER BY score DESC
LIMIT $limit
```
### HYBRID
Runs CONTENT_BASED and COLLABORATIVE separately, then combines their scores:
```
finalScore = (contentScore * SUGGESTION_HYBRID_CONTENT_WEIGHT) + (collaborativeScore * SUGGESTION_HYBRID_COLLABORATIVE_WEIGHT)
```
Default weights: `CONTENT_WEIGHT=0.6`, `COLLABORATIVE_WEIGHT=0.4`.
### SERENDIPITY
Recommends highly-rated movies from genres the user has **never** watched — intentional discovery outside the comfort zone.
```cypher
MATCH (u:User {id: $userId})-[:WATCHED]->(watched:Movie)-[:HAS_GENRE]->(g:Genre)
WITH u, COLLECT(DISTINCT g.name) AS knownGenres
MATCH (candidate:Movie)-[:HAS_GENRE]->(cg:Genre)
WHERE NOT (u)-[:WATCHED]->(candidate)
  AND NOT cg.name IN knownGenres
  AND candidate.imdbRating >= $serendipityMinRating
RETURN candidate ORDER BY candidate.imdbRating DESC
LIMIT $limit
```
---
## 12. REST Endpoints
| Method | Path | Description | Authorization |
|--------|------|-------------|---------------|
| `POST` | `/api/v1/users` | Create user | Public |
| `GET` | `/api/v1/users/{id}` | Get user and current profile | Admin or own user |
| `POST` | `/api/v1/users/{id}/watched` | Record watched movie `{movieId, rating}` → triggers AlgorithmSelector | Admin or own user |
| `POST` | `/api/v1/users/{id}/liked` | Record like `{movieId, suggestionAlgorithmUsed}` → triggers AlgorithmSelector | Admin or own user |
| `POST` | `/api/v1/users/{id}/disliked` | Record dislike `{movieId}` | Admin or own user |
| `GET` | `/api/v1/users/{id}/suggestions` | Suggestions for user. Query params: `?limit=10&algorithm=SERENDIPITY` (`algorithm` overrides profile) | Admin or own user |
| `GET` | `/api/v1/movies/{id}` | Get movie by ID | Authenticated |
| `POST` | `/api/v1/admin/import/trigger` | Invoke import Lambda `{searchTerms[], maxPages}` → returns 202 | Admin only |
| `GET` | `/metrics` | Prometheus metrics (port `METRICS_PORT`) | Internal |
| `GET` | `/api/v1/health` | Health check | Public |
### Authorization Rules
- All routes under `/api/v1/users/{id}/*` require: `role=admin` OR `token.sub == {id}`
- `/api/v1/admin/*` requires: `role=admin`
- Violations return `403 Forbidden`
- Missing/invalid token returns `401 Unauthorized`
---
## 13. Authentication Lambda (auth-lambda)
> This Lambda runs inside LocalStack locally and would run on AWS Lambda in production. It is the single source of truth for JWT tokens — the Go API never verifies tokens itself; it always delegates to this Lambda.
### handler.py
```python
from jwt_service import JwtService
import os
service = JwtService(
    secret=os.environ["JWT_SECRET"],
    expiration_hours=int(os.environ.get("JWT_EXPIRATION_HOURS", "24")),
    issuer=os.environ.get("JWT_ISSUER", "movie-suggestion-api")
)
def lambda_handler(event, context):
    action = event.get("action")
    if action == "generate":
        return service.generate(event["userId"], event["email"], event.get("role", "user"))
    if action == "validate":
        return service.validate(event["token"])
    return {"error": "unknown action"}
```
### jwt_service.py
```python
import jwt
from datetime import datetime, timedelta, timezone
class JwtService:
    def __init__(self, secret, expiration_hours, issuer):
        self.secret = secret
        self.expiration_hours = expiration_hours
        self.issuer = issuer
    def generate(self, user_id, email, role):
        now = datetime.now(timezone.utc)
        expires_at = now + timedelta(hours=self.expiration_hours)
        token = jwt.encode({
            "sub": user_id, "email": email, "role": role,
            "iss": self.issuer, "iat": now, "exp": expires_at
        }, self.secret, algorithm="HS256")
        return {"token": token, "expiresAt": expires_at.isoformat()}
    def validate(self, token):
        try:
            payload = jwt.decode(token, self.secret, algorithms=["HS256"],
                                 options={"require": ["sub", "email", "role", "exp"]})
            return {"valid": True, "userId": payload["sub"],
                    "email": payload["email"], "role": payload["role"]}
        except jwt.ExpiredSignatureError:
            return {"valid": False, "error": "token expired"}
        except jwt.InvalidTokenError as e:
            return {"valid": False, "error": str(e)}
```
### requirements.txt
```
PyJWT==2.8.0
cryptography==42.0.0
```
### Generate token (local development)
```bash
aws lambda invoke \
  --endpoint-url http://localhost:4566 \
  --function-name movie-suggestion-auth \
  --payload '{"action":"generate","userId":"123","email":"admin@example.com","role":"admin"}' \
  --cli-binary-format raw-in-base64-out \
  response.json && cat response.json
```
### auth.go middleware (API Go)
1. Extract Bearer token from `Authorization` header → `401` if missing
2. Invoke Lambda via AWS SDK with `action: "validate"`
3. If `valid=true` → inject `userId` and `role` into request context
4. If `valid=false` → return `401` with error message
5. Route handlers check `role=admin` OR `token.sub == pathParam {id}` → `403` if not authorized
---
## 14. Import Lambda (import-lambda)
> This Lambda is invoked asynchronously by the Go API. It searches OMDB for movies matching the given search terms and publishes one SQS message per movie found. The Go SQS consumer then processes each message independently.
### handler.py
```python
from omdb_client import OmdbClient
from sqs_publisher import SqsPublisher
import os, time
omdb = OmdbClient(os.environ["OMDB_API_URL"], os.environ["OMDB_API_KEY"],
                  int(os.environ.get("OMDB_REQUEST_TIMEOUT_SECONDS", "10")))
publisher = SqsPublisher(os.environ["SQS_QUEUE_URL"],
                         os.environ.get("AWS_ENDPOINT_URL"),
                         os.environ["AWS_REGION"])
rate_limit = float(os.environ.get("OMDB_RATE_LIMIT_RPS", "5"))
def lambda_handler(event, context):
    search_terms = event.get("searchTerms", [])
    max_pages = int(event.get("maxPages", 3))
    published, errors = 0, 0
    for term in search_terms:
        for page in range(1, max_pages + 1):
            results = omdb.search(term, page)
            if not results:
                break
            for movie in results:
                try:
                    publisher.publish(movie["imdbID"], movie["Title"])
                    published += 1
                except Exception:
                    errors += 1
            time.sleep(1.0 / rate_limit)
    return {"published": published, "errors": errors}
```
### omdb_client.py
```python
import requests
class OmdbClient:
    def __init__(self, base_url, api_key, timeout):
        self.base_url = base_url
        self.api_key = api_key
        self.timeout = timeout
    def search(self, term, page):
        resp = requests.get(self.base_url, params={
            "apikey": self.api_key, "s": term, "type": "movie", "page": page
        }, timeout=self.timeout)
        data = resp.json()
        return data.get("Search") if data.get("Response") == "True" else None
    def fetch_by_imdb_id(self, imdb_id):
        resp = requests.get(self.base_url, params={
            "apikey": self.api_key, "i": imdb_id, "plot": "full"
        }, timeout=self.timeout)
        return resp.json()
```
### sqs_publisher.py
```python
import boto3, json
from datetime import datetime, timezone
class SqsPublisher:
    def __init__(self, queue_url, endpoint_url, region):
        self.queue_url = queue_url
        self.client = boto3.client("sqs", endpoint_url=endpoint_url, region_name=region)
    def publish(self, imdb_id, title):
        self.client.send_message(
            QueueUrl=self.queue_url,
            MessageBody=json.dumps({
                "imdbId": imdb_id,
                "title": title,
                "requestedAt": datetime.now(timezone.utc).isoformat()
            })
        )
```
### requirements.txt
```
requests==2.31.0
boto3==1.34.0
```
---
## 15. SQS Consumer (API Go)
> The consumer runs as a background goroutine pool inside the Go API process. It is started in `main.go` alongside the HTTP server and is shut down gracefully on `SIGTERM`.
### Worker flow
1. Pool of `IMPORT_WORKER_COUNT` goroutines doing long polling
2. For each message:
   - a. Check if `imdbId` already exists in Neo4j → if yes, delete message and skip (idempotency layer 1)
   - b. Fetch full details from OMDB: `GET /?i={imdbId}&plot=full`
   - c. Upsert into Neo4j via `MERGE` (idempotency layer 2 — race condition protection)
   - d. Delete message from queue
   - e. On error: do NOT delete → SQS re-enqueues after visibility timeout → DLQ after 3 attempts
### infrastructure/lambda/import_client.go
Invokes import Lambda asynchronously (`InvocationType: Event`) via AWS SDK.
Does not wait for response — returns `202 Accepted` immediately to the admin.
---
## 16. Observability
### Prometheus Metrics (infrastructure/observability/metrics.go)
```
http_requests_total{method, path, status}              Counter
http_request_duration_seconds{method, path}            Histogram
suggestions_total{algorithm, userId}                   Counter
movie_import_total{status}                             Counter  # status: success, skipped, error
sqs_messages_processed_total{status}                   Counter
omdb_request_duration_seconds{status}                  Histogram
neo4j_query_duration_seconds{operation}                Histogram
```
### OpenTelemetry Tracing (infrastructure/observability/tracer.go)
- Initialize OTEL SDK exporting to Jaeger via OTLP gRPC
- HTTP middleware injects span per request with `correlationId`
- SQS consumer extracts `traceId` from message attribute to continue end-to-end trace
- Spans on: HTTP handlers, use cases, Neo4j repositories, OMDB calls, Lambda calls
### Logging
- Structured JSON with `zerolog`
- Fields: `level`, `timestamp`, `correlationId`, `traceId`, `spanId`, `message`
- `correlationId` generated by middleware and propagated via context
### Grafana Dashboard (scripts/grafana/dashboards/movie-suggestion-api.json)
| Panel | PromQL | Type |
|-------|--------|------|
| Requests/s by route | `rate(http_requests_total[5m])` | Time series |
| Latency p50/p95/p99 | `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))` | Time series |
| Error rate | `rate(http_requests_total{status=~"5.."}[5m])` | Time series |
| Suggestions by algorithm | `increase(suggestions_total[$__rate_interval])` | Bar chart |
| Movies imported | `increase(movie_import_total{status="success"}[$__rate_interval])` | Stat |
| Movies skipped (idempotency) | `increase(movie_import_total{status="skipped"}[$__rate_interval])` | Stat |
| SQS messages processed | `increase(sqs_messages_processed_total[$__rate_interval])` | Stat |
| OMDB latency p95 | `histogram_quantile(0.95, rate(omdb_request_duration_seconds_bucket[5m]))` | Time series |
| Neo4j latency by operation | `histogram_quantile(0.95, rate(neo4j_query_duration_seconds_bucket[5m]))` | Time series |
---
## 17. Environment Variables
```env
NEO4J_URI=bolt://localhost:7687
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=password
NEO4J_DATABASE=neo4j
OMDB_API_URL=http://www.omdbapi.com
OMDB_API_KEY=your_key_here
OMDB_REQUEST_TIMEOUT_SECONDS=10
SUGGESTION_DEFAULT_LIMIT=10
SUGGESTION_MAX_LIMIT=50
SUGGESTION_DEFAULT_ALGORITHM=POPULAR
SUGGESTION_HYBRID_CONTENT_WEIGHT=0.6
SUGGESTION_HYBRID_COLLABORATIVE_WEIGHT=0.4
SUGGESTION_MIN_IMDB_RATING=6.0
SUGGESTION_SERENDIPITY_MIN_RATING=7.5
SUGGESTION_CONTENT_BASED_MIN_WATCHES=5
SUGGESTION_COLLABORATIVE_MIN_WATCHES=20
SUGGESTION_CONTENT_PREFERENCE_THRESHOLD=0.7
SQS_QUEUE_URL=http://localhost:4566/000000000000/movie-import-queue
SQS_DLQ_URL=http://localhost:4566/000000000000/movie-import-dlq
SQS_VISIBILITY_TIMEOUT_SECONDS=30
SQS_WAIT_TIME_SECONDS=20
SQS_MAX_MESSAGES=10
IMPORT_WORKER_COUNT=5
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=test
AWS_SECRET_ACCESS_KEY=test
AWS_ENDPOINT_URL=http://localhost:4566
LAMBDA_AUTH_FUNCTION_NAME=movie-suggestion-auth
LAMBDA_IMPORT_FUNCTION_NAME=movie-suggestion-import
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_SERVICE_NAME=movie-suggestion-api
OTEL_TRACES_SAMPLER=always_on
METRICS_PORT=9091
METRICS_PATH=/metrics
SERVER_PORT=8080
SERVER_READ_TIMEOUT_SECONDS=30
SERVER_WRITE_TIMEOUT_SECONDS=30
LOG_LEVEL=info
LOG_FORMAT=json
JWT_SECRET=dev-secret
JWT_EXPIRATION_HOURS=24
JWT_ISSUER=movie-suggestion-api
```
---
## 18. scripts/aws/localstack-init.sh
> This script runs automatically when LocalStack starts (mounted as an init script). It creates SQS queues with Dead Letter Queue policy and deploys both Python Lambdas.
```bash
#!/bin/bash
set -e
/etc/localstack/init/ready.d/parameter-store.sh
awslocal sqs create-queue --queue-name movie-import-queue \
  --attributes VisibilityTimeout=30,MessageRetentionPeriod=86400
awslocal sqs create-queue --queue-name movie-import-dlq \
  --attributes MessageRetentionPeriod=1209600
DLQ_ARN=$(awslocal sqs get-queue-attributes \
  --queue-url http://localhost:4566/000000000000/movie-import-dlq \
  --attribute-names QueueArn --query Attributes.QueueArn --output text)
awslocal sqs set-queue-attributes \
  --queue-url http://localhost:4566/000000000000/movie-import-queue \
  --attributes "{\"RedrivePolicy\":\"{\\\"deadLetterTargetArn\\\":\\\"$DLQ_ARN\\\",\\\"maxReceiveCount\\\":\\\"3\\\"}\"}"
cd /tmp && cp -r /opt/auth-lambda . && cd auth-lambda
pip install -r requirements.txt -t . -q && zip -r function.zip . -q
awslocal lambda create-function \
  --function-name movie-suggestion-auth \
  --runtime python3.12 --handler handler.lambda_handler \
  --zip-file fileb:///tmp/auth-lambda/function.zip \
  --role arn:aws:iam::000000000000:role/lambda-role \
  --environment Variables="{JWT_SECRET=dev-secret,JWT_EXPIRATION_HOURS=24,JWT_ISSUER=movie-suggestion-api}"
cd /tmp && cp -r /opt/import-lambda . && cd import-lambda
pip install -r requirements.txt -t . -q && zip -r function.zip . -q
awslocal lambda create-function \
  --function-name movie-suggestion-import \
  --runtime python3.12 --handler handler.lambda_handler \
  --zip-file fileb:///tmp/import-lambda/function.zip \
  --role arn:aws:iam::000000000000:role/lambda-role \
  --environment Variables="{OMDB_API_KEY=your_key,OMDB_API_URL=http://www.omdbapi.com,SQS_QUEUE_URL=http://localstack:4566/000000000000/movie-import-queue,AWS_ENDPOINT_URL=http://localstack:4566,AWS_REGION=us-east-1,OMDB_RATE_LIMIT_RPS=5}"
echo ">>> LocalStack initialized"
```
---
## 19. docker-compose.yml
> Starts all infrastructure services. The Go API depends on `neo4j` (healthcheck) and `localstack` (healthcheck) being ready before starting. Observability stack (Prometheus, Grafana, Jaeger) runs in parallel.
```yaml
services:
  neo4j:
    image: neo4j:5
    container_name: movie_neo4j
    restart: unless-stopped
    environment:
      NEO4J_AUTH: neo4j/password
      NEO4J_PLUGINS: '["apoc"]'
    ports:
      - "7474:7474"
      - "7687:7687"
    volumes:
      - ./init/neo4j-init.cypher:/var/lib/neo4j/import/init.cypher:ro
      - neo4j_data:/data
    healthcheck:
      test: ["CMD-SHELL", "cypher-shell -u neo4j -p password 'RETURN 1'"]
      interval: 15s
      timeout: 10s
      retries: 10
      start_period: 30s
    networks:
      - movie-net
  localstack:
    image: localstack/localstack:3
    container_name: movie_localstack
    restart: unless-stopped
    ports:
      - "4566:4566"
    environment:
      AWS_ACCESS_KEY_ID: test
      AWS_SECRET_ACCESS_KEY: test
      SERVICES: lambda,sqs,ssm
      DEFAULT_REGION: us-east-1
      AWS_DEFAULT_REGION: us-east-1
      LOCALSTACK_HOST: localstack
    volumes:
      - ./scripts/aws/localstack-init.sh:/etc/localstack/init/ready.d/00-init.sh:ro
      - ./scripts/aws/parameter-store.sh:/etc/localstack/init/ready.d/parameter-store.sh:ro
      - ./auth-lambda:/opt/auth-lambda:ro
      - ./import-lambda:/opt/import-lambda:ro
      - localstack_data:/var/lib/localstack
    healthcheck:
      test: ["CMD-SHELL", "curl -sf http://localhost:4566/_localstack/health | grep '\"lambda\": *\"running\"'"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 30s
    networks:
      - movie-net
  movie-suggestion-api:
    build:
      context: .
      dockerfile: Dockerfile
    image: movie-suggestion:api
    restart: unless-stopped
    environment:
      NEO4J_URI: bolt://neo4j:7687
      NEO4J_USERNAME: neo4j
      NEO4J_PASSWORD: password
      SQS_QUEUE_URL: http://localstack:4566/000000000000/movie-import-queue
      SQS_DLQ_URL: http://localstack:4566/000000000000/movie-import-dlq
      AWS_ENDPOINT_URL: http://localstack:4566
      AWS_REGION: us-east-1
      AWS_ACCESS_KEY_ID: test
      AWS_SECRET_ACCESS_KEY: test
      LAMBDA_AUTH_FUNCTION_NAME: movie-suggestion-auth
      LAMBDA_IMPORT_FUNCTION_NAME: movie-suggestion-import
      OTEL_EXPORTER_OTLP_ENDPOINT: http://jaeger:4317
      OTEL_SERVICE_NAME: movie-suggestion-api
      METRICS_PORT: "9091"
      IMPORT_WORKER_COUNT: "5"
    ports:
      - "8080:8080"
      - "9091:9091"
    depends_on:
      neo4j:
        condition: service_healthy
      localstack:
        condition: service_healthy
    networks:
      - movie-net
  prometheus:
    image: prom/prometheus:v2.52.0
    container_name: movie_prometheus
    restart: unless-stopped
    ports:
      - "9090:9090"
    volumes:
      - ./scripts/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus_data:/prometheus
    command:
      - "--config.file=/etc/prometheus/prometheus.yml"
      - "--storage.tsdb.path=/prometheus"
      - "--storage.tsdb.retention.time=7d"
      - "--web.enable-lifecycle"
    networks:
      - movie-net
  grafana:
    image: grafana/grafana:11.0.0
    container_name: movie_grafana
    restart: unless-stopped
    ports:
      - "3000:3000"
    environment:
      GF_SECURITY_ADMIN_USER: admin
      GF_SECURITY_ADMIN_PASSWORD: admin
      GF_USERS_ALLOW_SIGN_UP: "false"
    volumes:
      - ./scripts/grafana/provisioning:/etc/grafana/provisioning:ro
      - ./scripts/grafana/dashboards:/var/lib/grafana/dashboards:ro
      - grafana_data:/var/lib/grafana
    depends_on:
      - prometheus
    networks:
      - movie-net
  jaeger:
    image: jaegertracing/all-in-one:1.57
    container_name: movie_jaeger
    restart: unless-stopped
    ports:
      - "16686:16686"
      - "4317:4317"
      - "4318:4318"
    environment:
      COLLECTOR_OTLP_ENABLED: "true"
    networks:
      - movie-net
volumes:
  neo4j_data:
  localstack_data:
  prometheus_data:
  grafana_data:
networks:
  movie-net:
    driver: bridge
```
---
## 20. scripts/prometheus/prometheus.yml
```yaml
global:
  scrape_interval: 15s
scrape_configs:
  - job_name: movie-suggestion-api
    static_configs:
      - targets: ["movie-suggestion-api:9091"]
```
---
## 21. Dockerfile (multi-stage)
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o movie-suggestion-api ./cmd/api
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/movie-suggestion-api .
EXPOSE 8080 9091
CMD ["./movie-suggestion-api"]
```
---
## 22. Best Practices
- All code in English (variables, comments, logs, error messages)
- Repositories: interfaces in `domain/`, implementations in `infrastructure/` — never the other way around
- Manual dependency injection in `main.go` — no DI framework; dependency graph must be explicit and readable
- `AlgorithmSelector` as pure domain service — no infrastructure dependency, easily unit-tested
- Typed domain errors: `ErrMovieNotFound`, `ErrUserNotFound`, `ErrAlgorithmNotFound`, `ErrUnauthorized`, `ErrForbidden`
- Structured JSON logging with `zerolog`, fields: `level`, `timestamp`, `correlationId`, `traceId`, `spanId`
- Graceful shutdown: 30s timeout, stop SQS consumer first, then drain HTTP connections, then close Neo4j driver
- Unit tests for `AlgorithmSelector` and use cases with mocks generated by `mockgen`
- README with: setup instructions, all environment variables, curl examples for all endpoints, how to generate tokens via Lambda
