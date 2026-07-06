# EXE-08 - Renomear endpoint de recomendacoes de GET /recommendations para GET /movies

Status: TODO
Prioridade: ALTA
Dependencias: EXE-02 concluida (estrutura de paginacao em recomendacoes)
Bloqueia: nenhuma

## Objetivo

Alterar o endpoint de recomendacoes de GET /api/v1/recommendations para GET /api/v1/movies, mantendo toda a estrutura atual de recomendacao.

## Resultado do Clarify (consolidado)

- Compatibilidade:
  - Hard cut: remover GET /api/v1/recommendations imediatamente.
- Rotas finais:
  - Manter coexistencia de GET /api/v1/movies (recomendacoes) e GET /api/v1/movies/{id} (detalhe).
- RBAC do novo GET /api/v1/movies (recomendacoes):
  - Permitir movies:read OU movies:write OU wildcard (*).
- Remocao de recommendations:read:
  - Remover do projeto todo (codigo, docs, testes e listas fixas de validacao).
- Contrato de query:
  - Manter limit, cursor, algorithm.
  - Incluir novo parametro title.
  - title: filtro parcial (contains), case-insensitive.
  - Semantica: algoritmo normal + filtro por title (intersecao/AND).
- Contrato de resposta:
  - Manter exatamente o payload atual de paginacao de recomendacoes (sem mudancas de schema).
- Documentacao:
  - Remover totalmente /recommendations do OpenAPI.
  - Atualizar README/curl examples para /movies.

## Regras obrigatorias desta tarefa

- Remover a regra recommendations:read deste fluxo.
- Manter somente movies:read para acesso ao endpoint de recomendacoes.
- Quem tem movies:read pode acessar a API de listagem de filmes recomendados.
- Manter exatamente a logica existente de recomendacao:
  - selecao de algoritmos
  - override por query parameter algorithm
  - paginacao cursor-based
  - metadados de resposta atuais
  - validacoes de limit/cursor/algorithm

## Quebra em micro-subtarefas (uma alteracao por item)

### Bloco A - Rotas e RBAC

ST-08-01
- Alterar a rota GET /api/v1/recommendations para GET /api/v1/movies no router.
- Arquivo: app/internal/infrastructure/http/router/router.go

ST-08-02
- Alterar o middleware dessa rota para permitir movies:read.
- Arquivo: app/internal/infrastructure/http/router/router.go

ST-08-03
- Garantir que movies:write e wildcard (*) tambem autorizem o endpoint de recomendacoes.
- Arquivo: app/internal/infrastructure/http/middleware/rbac.go (ou ponto equivalente de regra)

ST-08-04
- Validar que GET /api/v1/movies/{id} permanece registrado e funcional sem mudanca de comportamento.
- Arquivo: app/internal/infrastructure/http/router/router.go

### Bloco B - Contrato de query do endpoint

ST-08-05
- Adicionar parametro title no handler de recomendacoes.
- Arquivo: app/internal/infrastructure/http/handler/user_handler.go

ST-08-06
- Validar title como filtro opcional sem quebrar limit/cursor/algorithm.
- Arquivo: app/internal/infrastructure/http/handler/user_handler.go

ST-08-07
- Propagar title para o caso de uso de sugestoes (assinatura/entrada).
- Arquivo: app/internal/domain/usecase/suggest_movies.go

ST-08-08
- Ajustar implementacao do caso de uso para aceitar title e preservar logica atual de algoritmo.
- Arquivo: app/internal/application/usecase/suggest_movies_impl.go

ST-08-09
- Aplicar filtro title (contains, case-insensitive) dentro da consulta final do algoritmo (sem substituir algoritmo).
- Arquivo: app/internal/infrastructure/neo4j/recommendation_repository.go e/ou cypher associados

### Bloco C - Remocao de recommendations:read no projeto

ST-08-10
- Remover uso de recommendations:read no router e fluxos de autorizacao do endpoint de recomendacoes.
- Arquivos: app/internal/infrastructure/http/router/router.go, testes de middleware/handler

ST-08-11
- Remover recommendations:read de listas fixas de roles e validadores (se existirem).
- Arquivos: pontos de validacao de roles na aplicacao

ST-08-12
- Atualizar fixtures e seeds de teste para nao depender de recommendations:read.
- Arquivos: testes de integracao/unitarios afetados

### Bloco D - Testes unitarios

ST-08-13
- Atualizar testes do use case de sugestoes para novo parametro title.
- Arquivo: app/internal/application/usecase/suggest_movies_impl_test.go

ST-08-14
- Adicionar teste unitario para title com match parcial case-insensitive.
- Arquivo: app/internal/application/usecase/suggest_movies_impl_test.go

ST-08-15
- Adicionar teste unitario para semantica AND (algoritmo + filtro title).
- Arquivo: app/internal/application/usecase/suggest_movies_impl_test.go

### Bloco E - Testes de integracao HTTP

ST-08-16
- Migrar testes de paginacao de /recommendations para /movies.
- Arquivo: app/internal/infrastructure/http/handler/recommendations_pagination_integration_test.go (ou arquivo renomeado)

ST-08-17
- Atualizar testes de auth para exigir movies:read/movies:write/* no endpoint de recomendacoes.
- Arquivo: app/internal/infrastructure/http/handler/auth_recommendations_integration_test.go (ou arquivo renomeado)

ST-08-18
- Adicionar teste de 403 quando usuario nao possui movies:read nem movies:write nem *.
- Arquivo: app/internal/infrastructure/http/handler/*integration_test.go

ST-08-19
- Adicionar teste de filtro title no endpoint /movies com paginacao.
- Arquivo: app/internal/infrastructure/http/handler/*integration_test.go

### Bloco F - OpenAPI e README

ST-08-20
- Remover path /api/v1/recommendations do openapi.yaml.
- Arquivo: openapi.yaml

ST-08-21
- Definir GET /api/v1/movies como endpoint de recomendacoes com os parametros limit/cursor/algorithm/title.
- Arquivo: openapi.yaml

ST-08-22
- Manter GET /api/v1/movies/{id} como detalhe e diferenciar claramente as duas operacoes na descricao.
- Arquivo: openapi.yaml

ST-08-23
- Atualizar security requirements para refletir movies:read/movies:write/*.
- Arquivo: openapi.yaml

ST-08-24
- Atualizar README e exemplos curl para usar GET /api/v1/movies no fluxo de recomendacoes.
- Arquivo: README.md

### Bloco G - Validacao final

ST-08-25
- Rodar go test ./... e corrigir regressao restante.

ST-08-26
- Executar smoke manual dos endpoints:
  - GET /api/v1/movies (recomendacoes)
  - GET /api/v1/movies/{id} (detalhe)

ST-08-27
- Verificar ausencia de referencias residuais a /recommendations e recommendations:read no projeto.

## Checklist de pronto

- [ ] Rota GET /api/v1/movies ativa para recomendacoes
- [ ] Rota GET /api/v1/recommendations removida ou descontinuada conforme estrategia definida
- [ ] RBAC de recomendacoes usa apenas movies:read
- [ ] RBAC final aplicado: movies:read OU movies:write OU *
- [ ] Parametro title implementado (contains + case-insensitive)
- [ ] Algoritmos e paginacao preservados sem alteracao funcional
- [ ] recommendations:read removido de todo o projeto
- [ ] Testes de integracao atualizados e verdes
- [ ] openapi.yaml atualizado
- [ ] README atualizado para /movies
- [ ] go test ./... verde

## Evidencias esperadas

- Diff de rotas e middleware de autorizacao
- Resultado de testes de integracao de paginacao/recomendacao
- Trecho do openapi.yaml com GET /movies e GET /movies/{id}
