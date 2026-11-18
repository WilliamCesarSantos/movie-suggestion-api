# API Changes - Frontend Update Notes

## Objetivo
Este documento consolida as mudanças da API que impactam o frontend, incluindo alterações dos últimos dias e também mudanças feitas em dias anteriores dentro do histórico recente do projeto.

## Janela de atualização coberta
- 2026-07-02 até 2026-07-10
- Commits de referência: fd44d5f, 6f6462b, e2e9977, 0b22447, 2eb0510, 3bd4732, 8e1ba57, 1489436, c959221

## Resumo executivo
As mudanças mais relevantes para o frontend foram:
1. Migração do endpoint de recomendações de /api/v1/suggestions para /api/v1/movies.
2. Remoção do antigo GET /api/v1/movies de listagem de catálogo.
3. Inclusão de paginação por cursor no endpoint de recomendações.
4. Inclusão de GET /api/v1/users com regra de visibilidade por role.
5. Inclusão de PATCH /api/v1/users/{id} com regras de ownership.
6. Renomeação do endpoint de importação para /api/v1/movies-import.
7. Formalização de X-Correlation-ID e atualização do payload de health.

## Timeline detalhada por data

### 2026-07-02

#### 1) Novo endpoint GET /api/v1/users
- Adicionado endpoint de listagem de usuários com filtros e paginação.
- Query params suportados:
  - email
  - name
  - page
  - pageSize
- Regra de visibilidade:
  - usuário com apenas users:read visualiza apenas o próprio registro
  - usuário com users:write ou wildcard pode listar todos

Impacto no front:
- telas administrativas e de gestão de usuários podem usar paginação e filtros.
- UI deve tratar retorno paginado no formato data, total, page, pageSize.

#### 2) Recomendações com cursor pagination
- O fluxo de recomendações passou a aceitar cursor para paginação.
- Query params de recomendações:
  - limit
  - cursor
  - algorithm
- Resposta passou a incluir metadados de paginação por cursor.

Impacto no front:
- substituir paginação por página numérica por paginação baseada em nextCursor e prevCursor.
- persistir cursor da última resposta para navegação de páginas.

#### 3) Remoção do antigo GET /api/v1/movies de catálogo
- O endpoint de listagem de catálogo foi removido.
- GET /api/v1/movies/{id} foi mantido.

Impacto no front:
- qualquer tela que dependia de listagem global de filmes por catálogo precisa migrar para o fluxo de recomendações.

---

### 2026-07-05

#### 4) Novo endpoint PATCH /api/v1/users/{id}
- Inclusão de atualização parcial de usuário.
- Campos aceitos no body:
  - name
  - password
  - roles
- Regras importantes:
  - owner pode alterar name, password e roles
  - non-owner só pode alterar roles
  - non-owner alterando name ou password retorna 403

Impacto no front:
- tela de edição de usuário deve separar claramente campos permitidos por contexto.
- tratar 403 com mensagem específica de permissão.

#### 5) Migração de sugestões para recomendações em /api/v1/movies
- Endpoint antigo:
  - GET /api/v1/suggestions
- Endpoint novo:
  - GET /api/v1/movies
- Mudanças adicionais no mesmo fluxo:
  - operation e schema renomeados de Suggestions para Recommendations
  - adição de filtro title
  - acesso permitido para movies:read, movies:write ou wildcard
  - remoção da role suggestions:read do contrato

Impacto no front:
- atualizar todas as chamadas para usar /api/v1/movies no lugar de /api/v1/suggestions.
- revisar permissões no frontend: não usar mais suggestions:read.
- habilitar filtro por title quando aplicável.
- ajustar mapeamento de tipos para RecommendationsPageResponse.

---

### 2026-07-06

#### 6) Ajustes de login e consistência
- Atualizações de implementação no fluxo de login e consistência geral.
- No contrato OpenAPI, principal efeito foi correção estrutural de schema (sem mudança funcional relevante para payload esperado pelo front).

Impacto no front:
- sem quebra contratual relevante esperada.
- manter tratamento de erros 400, 401 e 500 no login.

---

### 2026-07-10

#### 7) Renomeação do endpoint de importação
- Endpoint antigo:
  - POST /api/v1/movie-import
- Endpoint novo:
  - POST /api/v1/movies-import
- Regra de autorização mantida:
  - movies:write

Impacto no front:
- atualizar URL de importação.
- sem mudança de payload no request/response da importação.

#### 8) Alinhamento OpenAPI com implementação real
- Health:
  - adição formal de resposta 503
  - payload ajustado para status, timestamp e services.neo4j
- Correlação:
  - X-Correlation-ID documentado como header de entrada
  - X-Correlation-ID documentado como header de saída
  - aplicado aos endpoints principais e respostas reutilizáveis
- Correção de exemplo de role em schema

Impacto no front:
- health check deve aceitar 200 e 503 como estados válidos de serviço disponível/degradado.
- se o frontend já usa correlation id, pode enviar X-Correlation-ID e também logar o valor retornado na resposta.

## Estado atual dos endpoints para o frontend

### Públicos
- GET /api/v1/health
- POST /api/v1/login

### Protegidos
- POST /api/v1/users
- GET /api/v1/users
- GET /api/v1/users/{id}
- PATCH /api/v1/users/{id}
- GET /api/v1/movies
- GET /api/v1/movies/{id}
- POST /api/v1/movies/{id}/watched
- POST /api/v1/movies-import

## Principais mudanças quebráveis para o frontend
1. Quebra de rota: /api/v1/suggestions foi substituída por /api/v1/movies.
2. Quebra de rota: /api/v1/movie-import foi substituída por /api/v1/movies-import.
3. Remoção de semântica antiga de GET /api/v1/movies como catálogo.
4. Mudança de estratégia de paginação em recomendações para cursor.
5. Alteração de role esperada no domínio de recomendações: remover suggestions:read e usar roles de movies.

## Checklist de ajuste no frontend
1. Buscar por referências antigas de endpoint e atualizar:
- /api/v1/suggestions
- /api/v1/movie-import

2. Revisar clientes/tipos de recomendações:
- usar RecommendationsPageResponse
- mapear nextCursor, prevCursor, hasNext, hasPrev, count, total

3. Revisar permissões em UI e guards:
- remover suposições sobre suggestions:read
- considerar movies:read e movies:write para acesso a recomendações

4. Ajustar telas de usuários:
- suportar GET /api/v1/users com filtros e paginação
- suportar PATCH /api/v1/users/{id} com regras de ownership

5. Ajustar health check:
- tratar 200 e 503
- ler formato atualizado do payload de health

6. Instrumentação opcional de correlação:
- enviar X-Correlation-ID nas requests
- registrar X-Correlation-ID retornado nas responses

## Observação final
Este arquivo é uma nota operacional para o frontend executar a migração de chamadas e contratos de forma segura, priorizando as mudanças quebráveis primeiro (rotas, paginação e permissões).