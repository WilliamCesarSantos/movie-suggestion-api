# Prompt de Alterações — Movie Suggestion App

## Objetivo

Eliminar as dependências das duas AWS Lambdas Python (`auth-lambda` e `import-lambda`), internalizando toda a lógica no serviço Go. Adicionar autenticação nativa com PostgreSQL e Argon2id (com pepper), controle de acesso baseado em RBAC, expor endpoint de importação de filmes diretamente na API, escalar a API com 3 réplicas atrás de Nginx, e fornecer um arquivo de demonstração completo das APIs.

---

## 1. Remoção da auth-lambda

### O que remover
- Diretório `auth-lambda/` completo (handler.py, jwt_service.py, requirements.txt, Dockerfile)
- `internal/infrastructure/lambda/auth_client.go`
- Todas as referências a `LambdaConfig.AuthFunctionName` e ao cliente `AuthClient` nos arquivos `config/config.go`, `cmd/api/main.go` e qualquer outro ponto de uso
- Na `docker-compose.yml`, remover os volumes e inicializações referentes à auth-lambda no LocalStack
- No `scripts/aws/localstack-init.sh`, remover o bloco de build e deploy da `auth-function`

### O que criar — serviço JWT interno em Go

Criar o pacote `internal/infrastructure/auth/` com dois arquivos:

**`jwt_service.go`** — Responsável por gerar e validar tokens JWT usando HMAC-SHA256 (HS256):
- `Generate(userID, email, role string) (token string, expiresAt time.Time, error)` — cria JWT com claims `sub` (userID), `email`, `role`, `iat`, `exp`
- `Validate(token string) (claims JWTClaims, error)` — valida assinatura e expiração, retorna os claims
- Configurado com `secret string` e `expiryHours int` lidos do `config.JWTConfig`
- Usar a biblioteca `github.com/golang-jwt/jwt/v5`

**`password_service.go`** — Responsável pelo hashing de senhas com Argon2id e pepper:
- `Hash(password string) (hash string, error)` — concatena `pepper + password`, gera hash Argon2id com salt aleatório, codificado no formato PHC string `$argon2id$v=19$m=65536,t=3,p=4$<salt_base64>$<hash_base64>`
- `Verify(password, hash string) (bool, error)` — concatena `pepper + password` antes de verificar contra o hash armazenado
- Configurado com `pepper string` lido de `config.AuthConfig.Pepper`
- Usar `golang.org/x/crypto/argon2`

O pepper nunca é armazenado no banco — ele protege contra vazamento de banco sem o valor do pepper.

### Adaptar o middleware de autenticação

- `internal/infrastructure/http/middleware/auth.go` — substituir a chamada ao `AuthClient` (Lambda) pela chamada ao `jwt_service.Validate(token)`
- Manter o comportamento atual: extrair `userId` e `role` do token e injetar no contexto

---

## 2. PostgreSQL para dados de autenticação

### docker-compose.yml

Adicionar serviço PostgreSQL:
```yaml
postgres:
  image: postgres:16
  environment:
    POSTGRES_DB: movie_suggestion
    POSTGRES_USER: postgres
    POSTGRES_PASSWORD: password
  ports:
    - "5432:5432"
  volumes:
    - postgres_data:/var/lib/postgresql/data
    - ./scripts/postgresql/init.sql:/docker-entrypoint-initdb.d/init.sql
```

Adicionar `postgres_data` ao bloco `volumes`. O serviço `api` deve declarar dependência de `postgres`.

### Script de inicialização SQL

Criar `scripts/postgresql/init.sql` (não em `init/`):
```sql
CREATE TABLE IF NOT EXISTS users (
    id          UUID PRIMARY KEY,
    email       VARCHAR(255) NOT NULL UNIQUE,
    name        VARCHAR(255) NOT NULL,
    password    TEXT NOT NULL,
    roles       TEXT[] NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Usuário inicial: william_cesar_santos@hotmail.com / 123456 / role: *
-- A senha abaixo é o hash Argon2id de (pepper="movie-suggestion-123456" + password="123456")
-- Gerar com: go run ./cmd/seed ou computar manualmente e substituir o placeholder abaixo
INSERT INTO users (id, email, name, password, roles, created_at)
VALUES (
    gen_random_uuid(),
    'william_cesar_santos@hotmail.com',
    'William',
    '$argon2id$v=19$m=65536,t=3,p=4$qNkRswLidbmSiP0zbdj81g$Y6hkfgo8OaAMoGT0hQLUlfFVWGjH2V2Tsv28qA2M0j4',
    ARRAY['*'],
    NOW()
) ON CONFLICT (email) DO NOTHING;
```

O hash acima foi computado com: `pepper="movie-suggestion-123456"` + `password="123456"` usando Argon2id (m=65536, t=3, p=4, salt=16 bytes, hash=32 bytes). Para regenerar: `go run ./cmd/seed -pepper movie-suggestion-123456 -password 123456`.

Manter o utilitário CLI `cmd/seed/main.go`:
1. Recebe `-pepper` e `-password` como flags
2. Imprime o hash PHC para uso em SQL ou testes

### Configuração

Adicionar ao `config/config.go` a struct:
```go
type PostgresConfig struct {
    DSN string // ex: "postgres://postgres:password@localhost:5432/movie_suggestion?sslmode=disable"
}

type AuthConfig struct {
    Pepper      string // variável de ambiente: ARGON2_PEPPER, default: "movie-suggestion-123456"
    ExpiryHours int    // variável de ambiente: JWT_EXPIRY_HOURS, default: 24
    Secret      string // variável de ambiente: JWT_SECRET, default: "dev-secret"
}
```

E campos `Postgres PostgresConfig` e `Auth AuthConfig` em `Config`. A `JWTConfig` existente deve ser absorvida por `AuthConfig`.

Adicionar variáveis no serviço `api` do `docker-compose.yml`:
```yaml
POSTGRES_DSN: postgres://postgres:password@postgres:5432/movie_suggestion?sslmode=disable
JWT_EXPIRY_HOURS: "24"
ARGON2_PEPPER: "movie-suggestion-123456"
```

### ORM — GORM para PostgreSQL

Usar `gorm.io/gorm` com o driver `gorm.io/driver/postgres` (que usa `github.com/jackc/pgx/v5` internamente via pgx DSN).

Criar o model GORM em `internal/infrastructure/postgres/model/auth_user.go`:
```go
type AuthUserModel struct {
    ID        string         `gorm:"type:uuid;primaryKey"`
    Email     string         `gorm:"uniqueIndex;not null"`
    Name      string         `gorm:"not null"`
    Password  string         `gorm:"not null"`
    Roles     pq.StringArray `gorm:"type:text[];not null"`
    CreatedAt time.Time
}

func (AuthUserModel) TableName() string { return "users" }
```

Não usar `gorm.AutoMigrate` — o schema é gerenciado pelo `scripts/postgresql/init.sql`.

Criar `internal/infrastructure/postgres/user_repository.go`:
- Struct `authUserRepository` com `*gorm.DB`
- Implementar a interface do domínio:
  ```go
  type AuthUserRepository interface {
      Create(ctx context.Context, user *entity.AuthUser) error
      FindByEmail(ctx context.Context, email string) (*entity.AuthUser, error)
  }
  ```
- Converter entre `entity.AuthUser` (domínio) e `AuthUserModel` (infraestrutura) — nunca expor o model GORM para fora do pacote `postgres/`
- Entidade `entity.AuthUser` em `internal/domain/entity/auth_user.go`:
  ```go
  type AuthUser struct {
      ID        string
      Name      string
      Email     string
      Password  string   // hash Argon2id com pepper, nunca a senha em texto claro
      Roles     []string
      CreatedAt time.Time
  }
  ```

Dependências a adicionar:
```
gorm.io/gorm
gorm.io/driver/postgres
github.com/lib/pq        // para pq.StringArray (tipo array PostgreSQL)
```

Remover:
```
github.com/jackc/pgx/v5  // se não usado diretamente fora do GORM
```

---

## 3. Endpoint de cadastro de usuário

### `POST /api/v1/users` — requer autenticação

O endpoint **não é público**. Requer Bearer token com role `users:write` (sem restrição de identidade — qualquer usuário autenticado com essa role pode criar outros usuários).

O endpoint existente `CreateUser` em `internal/infrastructure/http/handler/user_handler.go` deve ser atualizado para aceitar o novo body:
```json
{
  "name": "Alice",
  "email": "alice@example.com",
  "password": "s3cr3t",
  "roles": ["users:read", "users:write", "suggestions:read", "movies:read"]
}
```

`roles` é **obrigatório** e deve conter ao menos um elemento. Retornar `400 Bad Request` se ausente ou vazio.

**Comportamento transacional:**
1. Validar que `email`, `name`, `password` e `roles` (não vazio) foram fornecidos
2. Gerar um UUID para o usuário
3. Fazer hash da senha com Argon2id via `PasswordService.Hash` (pepper + password)
4. Salvar no PostgreSQL via `AuthUserRepository.Create` (id, name, email, hash, roles, created_at)
5. Salvar no Neo4j via `UserRepository` existente (somente id, name, email — sem senha, sem roles)
6. Retornar `201 Created` com o body do usuário (sem expor senha ou hash):
   ```json
   { "id": "...", "name": "Alice", "email": "alice@example.com", "roles": ["users:read"], "createdAt": "..." }
   ```

Se o email já existir no PostgreSQL, retornar `409 Conflict`.

No router, este endpoint deve ter o middleware de autenticação + `RequireRole("users:write")` (sem `RequireOwnerOrWildcard`).

---

## 4. Novo endpoint de login

### `POST /api/v1/login`

Criar handler em `internal/infrastructure/http/handler/auth_handler.go`:

Request body:
```json
{ "email": "alice@example.com", "password": "s3cr3t" }
```

**Comportamento:**
1. Buscar `AuthUser` pelo email no PostgreSQL via `AuthUserRepository.FindByEmail`
2. Verificar senha com `PasswordService.Verify(input.password, authUser.Password)` (pepper é aplicado internamente)
3. Se inválido, retornar `401 Unauthorized`
4. Gerar JWT via `JWTService.Generate(authUser.ID, authUser.Email, authUser.Roles)`
5. Retornar `200 OK` — **não retornar `userId` na resposta**:
   ```json
   { "token": "...", "email": "...", "roles": ["users:read"], "expiresAt": "..." }
   ```

O JWT deve carregar nos claims: `sub` (userID), `email`, `roles` (array de strings), `iat`, `exp`.

Este endpoint não requer autenticação (sem middleware de auth).

Criar use case de login em `internal/domain/usecase/login.go` (interface) e `internal/application/usecase/login_impl.go` (implementação):
```go
type LoginUseCase interface {
    Execute(ctx context.Context, email, password string) (*LoginResult, error)
}

type LoginResult struct {
    Token     string
    Email     string
    Roles     []string
    ExpiresAt time.Time
}
```

Registrar a rota no router (`internal/infrastructure/http/router/router.go`) sem middleware de autenticação.

---

## 5. Remoção da import-lambda

### O que remover
- Diretório `import-lambda/` completo (handler.py, omdb_client.py, sqs_publisher.py, requirements.txt, Dockerfile)
- `internal/infrastructure/lambda/import_client.go`
- `internal/application/usecase/import_movies_impl.go` (substituir completamente)
- Referências a `LambdaConfig.ImportFunctionName` em `config/config.go` e `cmd/api/main.go`
- No `scripts/aws/localstack-init.sh`, remover o bloco de build e deploy da `import-function`
- Se após remoção o pacote `internal/infrastructure/lambda/` ficar vazio, removê-lo
- Remover endpoints `POST /api/v1/users/{id}/liked` e `POST /api/v1/users/{id}/disliked` (handlers, rotas e use cases correspondentes)

### Simplificar LocalStack

Com as duas lambdas removidas, o LocalStack só precisa do serviço SQS. Atualizar `docker-compose.yml`:
```yaml
SERVICES: sqs,ssm
```

---

## 6. Endpoint de importação de filmes

### `POST /api/v1/movie-import`

Criar handler em `internal/infrastructure/http/handler/import_handler.go`:

Request body:
```json
{
  "searchTerms": ["inception", "matrix"],
  "maxPages": 2
}
```

A chave OMDB é configurada exclusivamente via variável de ambiente `OMDB_API_KEY` (ou argumento da aplicação) — nunca via parâmetro de API.

Resposta imediata: `202 Accepted`
```json
{ "status": "import triggered" }
```

Requer autenticação com role `movies:write` (middleware RBAC).

**Novo use case `ImportMoviesUseCase`** em `internal/domain/usecase/import_movies.go` (interface) e `internal/application/usecase/import_movies_impl.go` (implementação):

```go
type ImportMoviesUseCase interface {
    Execute(ctx context.Context, searchTerms []string, maxPages int) error
}
```

**Implementação** — executar em goroutine (fire-and-forget após retornar 202):
1. Para cada `term` e cada `page` de 1 a `maxPages`:
   a. Chamar `OmdbSearcher.Search(ctx, term, page)` — retorna `[]SearchResult{ImdbID, Title}`
   b. Para cada `SearchResult`, publicar `{"imdbId": "tt1234567"}` na fila SQS via `MovieImportPublisher`
3. Erros de busca devem ser logados, não devem impedir o processamento dos demais termos

### Novo SQS Publisher

Criar `internal/infrastructure/sqs/publisher.go`:
```go
type Publisher struct {
    client   *sqs.Client
    queueURL string
}

func (p *Publisher) Publish(ctx context.Context, imdbID string) error
```

O `Consumer` existente e o `ProcessMovieImportUseCase` existente já tratam a mensagem recebida da fila e salvam o filme no Neo4j — **não alterar esse fluxo**.

### Endpoint `/api/v1/movie/{id}/watched`

Substituir `POST /api/v1/users/{id}/watched` por `POST /api/v1/movie/{id}/watched`, onde `{id}` é o **ID do filme** (não do usuário).

O ID do usuário é extraído do claim `sub` do JWT (não da URL). Requer role `movie-watch:write` (sem restrição de identidade por URL).

Request body:
```json
{
  "rating": 8.5,
  "reaction": "liked"
}
```

- `rating` — opcional, float64, pontuação pessoal do usuário (0.0–10.0). Armazenada na relação `WATCHED` no Neo4j como propriedade `userRating`
- `reaction` — opcional, string enum: `"liked"` | `"disliked"`. Se omitido, não registra reação. Armazenada na relação `WATCHED` como propriedade `reaction`

Comportamento:
1. Extrair `userID` do claim `sub` do JWT (via contexto)
2. Buscar filme por `{id}` no Neo4j — retornar `404` se não encontrado
3. Criar ou atualizar a relação `(User)-[:WATCHED {userRating, reaction, watchedAt}]->(Movie)` no Neo4j
4. Atualizar o `watchCount` do usuário no Neo4j
5. Retornar `200 OK` com a relação criada

O use case `ManageUserUseCase.RecordWatched` deve ser atualizado para aceitar os novos campos `userRating float64` e `reaction string`. Os endpoints `/liked` e `/disliked` devem ser **removidos** — a reação é registrada junto com o watched.

### Idempotência dos filmes no Neo4j

O `Upsert` existente em `internal/infrastructure/neo4j/movie_repository.go` já usa `MERGE (m:Movie {imdbId: $imdbId})`, garantindo idempotência pelo campo `imdbId`. O campo `id` (UUID interno) deve ser definido na primeira inserção e preservado nos merges subsequentes. Garantir no Cypher:
```cypher
MERGE (m:Movie {imdbId: $imdbId})
ON CREATE SET m.id = $id, m.createdAt = datetime()
SET m.title = $title, m.year = $year, ...
```

---

## 7. Revisão da estrutura — Clean Architecture

Rever toda a estrutura do projeto aplicando os princípios de Clean Architecture de forma rigorosa. A regra fundamental: dependências apontam sempre para dentro (Domain ← Application ← Infrastructure).

### Camadas e responsabilidades

```
internal/
├── domain/                   ← núcleo, zero dependências externas
│   ├── entity/               ← entidades de negócio (User, Movie, AuthUser, Suggestion)
│   ├── repository/           ← interfaces de repositório (contratos)
│   └── usecase/              ← interfaces de use case (contratos)
├── application/              ← orquestração, depende apenas de domain
│   └── usecase/              ← implementações dos use cases
└── infrastructure/           ← implementações concretas, depende de domain
    ├── auth/                 ← JWTService, PasswordService
    ├── http/
    │   ├── handler/          ← handlers HTTP (depende de domain/usecase interfaces)
    │   ├── middleware/       ← auth, RBAC, observability
    │   └── router/           ← registro de rotas
    ├── neo4j/                ← implementações dos repositórios Neo4j
    │   └── cypher/           ← queries Cypher constantes
    ├── postgres/             ← implementações dos repositórios PostgreSQL
    ├── observability/        ← metrics, tracer
    ├── omdb/                 ← cliente OMDB
    └── sqs/                  ← consumer e publisher SQS
```

### Problemas a corrigir

1. **`MovieHandler` importa `repository.MovieRepository` diretamente** — violar Clean Arch. Criar use case `GetMovieUseCase` no domain e mover a lógica de busca para application. O handler deve depender apenas da interface do use case.

2. **`importMoviesUseCase` importa `infrastructure/lambda`** — já será corrigido com a remoção da lambda. A nova implementação deve depender de interfaces de domínio (`OmdbSearcher`, `MovieImportPublisher`) injetadas, nunca de structs concretas de infra.

3. **Interfaces de infra no domain** — as interfaces `OmdbSearcher` e `MovieImportPublisher` necessárias ao `ImportMoviesUseCase` devem ser declaradas em `internal/domain/usecase/import_movies.go` (dentro do mesmo arquivo da interface ou em arquivo separado no mesmo pacote):
   ```go
   type OmdbSearcher interface {
       Search(ctx context.Context, term string, page int) ([]SearchResult, error)
   }
   type MovieImportPublisher interface {
       Publish(ctx context.Context, imdbID string) error
   }
   ```

4. **`ProcessMovieImportUseCase`** — já correto (depende de interfaces de repositório e do client OMDB via interface).

5. **Handlers não devem depender de structs concretas de repositório** — todos os handlers devem receber interfaces de use case como dependência.

---

## 8. Controle de acesso RBAC

### Modelo de roles

Cada usuário possui um array de roles (`[]string`). A role `*` concede acesso a todos os endpoints sem exceção.

### Regras por endpoint

| Método | Path | Role necessária |
|--------|------|-----------------|
| `GET` | `/api/v1/health` | nenhuma |
| `POST` | `/api/v1/login` | nenhuma |
| `POST` | `/api/v1/users` | `users:write` (sem restrição de identidade) |
| `GET` | `/api/v1/users/{id}` | `users:read` **+ restrição de identidade** |
| `POST` | `/api/v1/movie/{id}/watched` | `movie-watch:write` (user do token, sem restrição de URL) |
| `GET` | `/api/v1/users/{id}/suggestions` | `suggestions:read` **+ restrição de identidade** |
| `GET` | `/api/v1/movies/{id}` | `movies:read` |
| `POST` | `/api/v1/movie-import` | `movies:write` |
| `GET` | `/metrics` | nenhuma (porta 9090) |

### Restrição de identidade

Para os endpoints de usuário (`/users/{id}/*`), além da role, deve ser verificado que o `{id}` da URL corresponde ao `sub` do token JWT, **exceto** quando o usuário possui a role `*`. Ou seja:

- Usuário com role `*` → passa em qualquer endpoint sem verificação de identidade
- Usuário sem role `*` → deve ter a role do endpoint E `{id}` == `sub` do token

Para o endpoint `GET /api/v1/users/{id}/suggestions`, usar o `sub` do token como ID do usuário para buscar sugestões, **ignorando o `{id}` da URL** se eles divergirem (ou retornar 403 — escolher consistência: retornar `403 Forbidden` se `{id}` != `sub` e role != `*`).

### Implementação do middleware RBAC

Criar `internal/infrastructure/http/middleware/rbac.go`:

```go
// RequireRole retorna um middleware que exige que o usuário autenticado possua
// a role especificada, ou a role curinga "*".
func RequireRole(role string) func(http.Handler) http.Handler

// RequireOwnerOrWildcard verifica que o {id} da URL corresponde ao sub do token,
// ou que o usuário possui a role "*".
func RequireOwnerOrWildcard() func(http.Handler) http.Handler
```

O middleware de autenticação deve extrair do JWT o array completo de roles (`[]string`) e injetá-lo no contexto, não apenas uma string. Atualizar `middleware/auth.go`:
- Chave de contexto `ContextKeyRoles` do tipo `[]string` (substituir `ContextKeyRole` que era uma única string)
- `ContextKeyUserID` permanece igual

O `JWTClaims` deve carregar `Roles []string` (claim `roles` no JWT).

### Roles do usuário inicial

O usuário inicial `william_cesar_santos@hotmail.com` deve ter `roles = ARRAY['*']` no PostgreSQL.

---

## 9. Atualizar docker-compose.yml — réplicas, Nginx e build

### Serviço api — build e réplicas

O serviço `api` deve ser construído a partir do `Dockerfile` local e executado em 3 réplicas. Remover o mapeamento de porta direta da API (quem expõe a porta é o Nginx):

```yaml
api:
  build: .
  deploy:
    replicas: 3
  environment:
    # ... (sem port mapping aqui)
  depends_on:
    - neo4j
    - localstack
    - jaeger
    - postgres
```

### Serviço Nginx

Criar `nginx/nginx.conf` e adicionar serviço ao `docker-compose.yml`:

```yaml
nginx:
  image: nginx:1.27-alpine
  ports:
    - "8080:80"
  volumes:
    - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
  depends_on:
    - api
```

Conteúdo de `nginx/nginx.conf`:
```nginx
events { worker_connections 1024; }

http {
  upstream api {
    server api:8080;
  }

  server {
    listen 80;

    location / {
      proxy_pass         http://api;
      proxy_set_header   Host $host;
      proxy_set_header   X-Real-IP $remote_addr;
      proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
    }
  }
}
```

O Docker Compose com `replicas: 3` e a diretiva `server api:8080` no upstream do Nginx usa o DNS interno do Docker para resolver para qualquer réplica disponível, distribuindo as requisições automaticamente.

A porta de métricas `9090` não passa pelo Nginx — acessar diretamente em cada réplica não é necessário para o demo. Remover o mapeamento `9090:9090` do serviço `api`.

---

## 10. Atualizar config/config.go

- Remover `LambdaConfig` completamente
- Remover `JWTConfig` (absorvida por `AuthConfig`)
- Adicionar `PostgresConfig` e `AuthConfig` conforme seção 2
- Variáveis de ambiente a ler:
  - `POSTGRES_DSN` → `PostgresConfig.DSN`
  - `ARGON2_PEPPER` → `AuthConfig.Pepper` (default: `"movie-suggestion-123456"`)
  - `JWT_SECRET` → `AuthConfig.Secret` (default: `"dev-secret"`)
  - `JWT_EXPIRY_HOURS` → `AuthConfig.ExpiryHours` (default: `24`)

---

## 11. Atualizar cmd/api/main.go

- Remover inicialização do Lambda client (AWS SDK Lambda)
- Inicializar pool de conexões PostgreSQL (`pgxpool.New`)
- Inicializar `AuthUserRepository` (postgres)
- Inicializar `PasswordService` (com pepper de `cfg.Auth.Pepper`)
- Inicializar `JWTService` (com secret e expiryHours de `cfg.Auth`)
- Inicializar `LoginUseCase`
- Inicializar `GetMovieUseCase`
- Inicializar `SQSPublisher`
- Inicializar `ImportMoviesUseCase` com o OMDB client (interface `OmdbSearcher`) e o publisher (interface `MovieImportPublisher`)
- Registrar rotas novas no router: `POST /api/v1/login` e `POST /api/v1/import`
- Passar `JWTService` para o middleware de autenticação (substituindo `AuthClient`)
- Criar utilitário `cmd/seed/main.go` para gerar hash Argon2id de senha + pepper

---

## 12. Dependências Go a adicionar (go.mod)

```
github.com/golang-jwt/jwt/v5
gorm.io/gorm
gorm.io/driver/postgres
github.com/lib/pq
golang.org/x/crypto
```

Remover:
```
github.com/aws/aws-sdk-go-v2/service/lambda
github.com/jackc/pgx/v5   // substituído pelo driver do GORM
```

---

## 13. Arquivo de demonstração

Criar `scripts/demo.sh` — script Bash que demonstra o uso completo de todas as APIs em sequência:

```
1.  Aguardar a API estar disponível (health check em loop)
2.  Login com o usuário inicial (william / 123456) → extrair TOKEN_ADMIN
3.  Criar novo usuário Alice com TOKEN_ADMIN (POST /users, roles=[users:read,users:write,suggestions:read,movies:read,movie-watch:write])
4.  Login como Alice → extrair TOKEN_ALICE
5.  Disparar importação de filmes (TOKEN_ADMIN, POST /movie-import):
      searchTerms=["inception","matrix"], maxPages=1
6.  Aguardar 10s para o SQS consumer processar
7.  Buscar usuário Alice com TOKEN_ALICE (GET /users/{alice_id})
8.  Tentar buscar usuário Alice com TOKEN_ADMIN (deve funcionar por role *)
9.  Registrar filme assistido como Alice (POST /movie/{movie_id}/watched) com rating=8.5, reaction="liked"
10. Registrar outro filme assistido com reaction="disliked" e rating=4.0
11. Buscar sugestões de filmes para Alice (GET /users/{alice_id}/suggestions)
12. Buscar sugestões com algoritmo SERENDIPITY
13. Buscar detalhes de um filme (GET /movies/{movie_id})
14. Tentar acessar /movie/{id}/watched com token inválido → esperar 401
15. Tentar criar usuário sem token (POST /users sem auth) → esperar 401
16. Tentar criar usuário com TOKEN_ALICE (sem role users:write) → esperar 403
17. Health check final
```

O script deve usar `jq` para extrair valores das respostas JSON, imprimir cada passo com separador visual, e abortar com mensagem de erro em caso de falha inesperada. Usar variáveis para BASE_URL, email, senha e IDs extraídos dinamicamente.

---

## Resumo das rotas após as alterações

| Método | Path | Auth | Role | Descrição |
|--------|------|------|------|-----------|
| `GET` | `/api/v1/health` | Nenhuma | — | Health check |
| `POST` | `/api/v1/login` | Nenhuma | — | Autenticar (email+senha → JWT) |
| `POST` | `/api/v1/users` | Bearer | `users:write` | Cadastrar usuário (sem restrição de identidade) |
| `GET` | `/api/v1/users/{id}` | Bearer | `users:read` + identidade | Buscar usuário |
| `POST` | `/api/v1/movie/{id}/watched` | Bearer | `movie-watch:write` | Registrar assistido + reação + nota (user do token) |
| `GET` | `/api/v1/users/{id}/suggestions` | Bearer | `suggestions:read` + identidade | Sugestões (usa sub do token) |
| `GET` | `/api/v1/movies/{id}` | Bearer | `movies:read` | Buscar filme |
| `POST` | `/api/v1/movie-import` | Bearer | `movies:write` | Disparar importação |
| `GET` | `/metrics` | Nenhuma (porta 9090) | — | Métricas Prometheus |
