# Prompt de Alterações

## Contexto

Este é um projeto em Go de sugestão de filmes com arquitetura hexagonal (Clean Architecture). Atualmente o projeto realiza injeção de dependência de forma manual no `cmd/api/main.go`, instanciando e conectando todos os componentes explicitamente. A estrutura de pastas mistura código-fonte com scripts de infraestrutura local na raiz do repositório.

## Alterações Solicitadas

### 1. Injeção de Dependência com `uber-go/fx`

Substitua a injeção de dependência manual do `cmd/api/main.go` pelo framework `uber-go/fx`.

- Adicione a dependência `go.uber.org/fx` ao `go.mod` via `go get go.uber.org/fx`
- Crie módulos `fx.Module` por camada:
  - `config` — provê `*config.Config` a partir de `config.Load()`
  - `infrastructure` — provê os drivers (Neo4j, Postgres, AWS/SQS), repositórios, serviços de auth, cliente OMDB e observabilidade
  - `application` — provê os use cases e o `AlgorithmSelector`/`AlgorithmDispatcher`
  - `http` — provê handlers, middlewares e o router
- Substitua o `main()` por uma invocação `fx.New(...)` com os módulos acima e o lifecycle do servidor HTTP usando `fx.Hook` (OnStart/OnStop)
- O consumer SQS deve ser registrado no lifecycle do `fx.App` também
- Mantenha o graceful shutdown existente, agora gerenciado pelo próprio `fx`

### 2. Reestruturação de Pastas

#### 2.1 Criar pasta `app` para o código-fonte

Mova todos os arquivos de código-fonte Go para dentro de `app/`, preservando a estrutura interna:

```
app/
  cmd/
    api/
      main.go
    seed/
      main.go
  config/
    config.go
  internal/
    application/
      suggestion/
      usecase/
    domain/
      entity/
      repository/
      usecase/
    infrastructure/
      auth/
      http/
      neo4j/
      observability/
      omdb/
      postgres/
      sqs/
```

- Atualize todos os `import paths` no `go.mod` (module path) e em todos os arquivos `.go` para refletir os novos caminhos
- Mantenha `go.mod` e `go.sum` na raiz do repositório (o Go module root permanece na raiz)

#### 2.2 Mover o `Dockerfile` para `app/docker/`

- Mova `Dockerfile` para `app/docker/Dockerfile`
- Ajuste os caminhos de `COPY` e contexto de build dentro do `Dockerfile` para a nova estrutura
- Atualize o `build context` no `docker-compose.yml` (campo `build:`) para apontar para `app/docker/` ou use `dockerfile: app/docker/Dockerfile`

#### 2.3 Renomear `scripts` para `local`

- Renomeie a pasta `scripts/` para `local/`
- Mova o arquivo `docker-compose.yml` da raiz para dentro de `local/`
- Atualize todos os caminhos de volumes e binds no `docker-compose.yml` para refletir que ele agora está em `local/` (os caminhos relativos precisam subir um nível: `../` para acessar a raiz)
- Atualize referências ao `docker-compose.yml` no `README.md` e em quaisquer scripts que o referenciem

### 3. Estrutura Final Esperada

```
app/
  cmd/
  config/
  docker/
    Dockerfile
  internal/
local/
  aws/
    localstack-init.sh
    parameter-store.sh
  grafana/
    dashboards/
    provisioning/
  neo4j/
    init.cypher
  nginx/
    nginx.conf
  postgresql/
    init.sql
  prometheus/
    prometheus.yml
  demo.sh
  docker-compose.yml
go.mod
go.sum
README.md
changes.md
```

### Restrições e Observações

- Não altere a lógica de negócio existente (domain, use cases, repositórios)
- Mantenha todos os arquivos de teste (`*_test.go`) junto ao código que testam, dentro de `app/`
- O module path do `go.mod` deve permanecer `github.com/WilliamCesarSantos/movie-suggestion`; apenas os import paths internos mudam para incluir o prefixo `app/` quando necessário
- Certifique-se de que `go build ./...` e `go test ./...` passem após as alterações
