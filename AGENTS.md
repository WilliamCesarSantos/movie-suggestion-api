# AGENTS.md

Este arquivo resume o contexto essencial do projeto para agentes, reduzindo buscas repetitivas e consumo de tokens.

## Projeto

- Nome: Movie Suggestion API
- Linguagem: Go
- Arquitetura: Clean Architecture
- Modulo Go: github.com/WilliamCesarSantos/movie-suggestion-api

## Estrutura principal

- app/cmd/api/main.go: bootstrap e DI
- app/config/config.go: configuracao por env
- app/internal/domain: entidades + contratos
- app/internal/application: casos de uso e regras
- app/internal/infrastructure: adapters HTTP, auth, Neo4j, Postgres, SQS, observabilidade
- app/internal/infrastructure/http/router/router.go: roteamento
- app/internal/infrastructure/http/middleware: auth, RBAC, observabilidade
- app/internal/infrastructure/http/handler: handlers HTTP
- openapi.yaml: contrato da API
- local/docker-compose.yml: ambiente local

## Estado funcional atual relevante

- Sugestoes usam email do token no handler/use case; identificacao interna usa ID resolvido em repositorio.
- Middleware de auth injeta no contexto:
  - userId
  - userEmail
  - roles
- Foi adicionado teste de integracao para fluxo login -> suggestions validando uso do email do token.

## Decisoes refinadas (fonte de verdade)

Arquivos de planejamento:
- .github/tasks/refinement-backlog.md
- .github/tasks/execution/README.md
- .github/tasks/execution/task-01-list-users.md
- .github/tasks/execution/task-02-suggestions-pagination.md
- .github/tasks/execution/task-03-remove-get-movies.md
- .github/tasks/execution/task-04-openapi-update.md
- .github/tasks/execution/task-05-unused-code-cleanup.md
- .github/tasks/execution/task-06-logging-correlation-pod.md

Resumo das decisoes de clarify:
- Novo GET /api/v1/users com regra por role (users:read apenas ve a si mesmo).
- /suggestions com paginacao cursor-based e cursor opaco assinado.
- Remover GET /api/v1/movies.
- Atualizar somente openapi.yaml para refletir mudancas.
- Limpeza de codigo sem uso inclui itens usados apenas por testes (preservar artefatos de desenvolvimento).
- Logs com correlationId ponta a ponta, propagacao em SQS por MessageAttributes e pod via hostname.

## Comandos rapidos

- Rodar testes: go test ./...
- Rodar API local: go run ./app/cmd/api
- Subir stack local: docker compose -f local/docker-compose.yml up -d

## Convencoes de implementacao

- Preservar separacao por camadas (domain/application/infrastructure).
- Evitar logica de regra de negocio em handler.
- Priorizar testes de use case e integracao HTTP para mudancas de contrato.
- Sempre alinhar mudancas de endpoint com openapi.yaml.
- Em mudancas de auth, manter coerencia entre claims JWT e contexto do middleware.

## Checklist minimo antes de concluir tarefa

- Build e testes verdes: go test ./...
- Contratos HTTP consistentes com implementacao
- Sem regressao de RBAC/auth
- Evidencias no arquivo de tarefa de execucao correspondente
