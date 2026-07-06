# EXE-07 - PATCH /users/{id} para atualizar name, password e roles

Status: TODO
Prioridade: ALTA
Dependencias: EXE-01 (usuarios), EXE-04 (openapi pode ser atualizado nesta task)
Bloqueia: nenhuma

## Objetivo

Implementar PATCH /api/v1/users/{id} com atualizacao parcial de usuario, incluindo validacoes de autorizacao e de payload conforme clarify.

## Definicoes consolidadas do clarify

- Endpoint: PATCH /api/v1/users/{id}
- Acesso: users:write ou wildcard (*)
- Regras de alteracao:
  - owner (token id igual ao path id): pode alterar name, password e roles
  - nao-owner: pode alterar somente roles
  - nao-owner tentando alterar name/password: 403 e bloqueia a requisicao inteira
- Roles:
  - substitui lista inteira
  - [] permitido
  - validar contra lista fixa permitida
  - lista permitida: users:read, users:write, recommendations:read, movies:read, movies-watch:write, movies:write
- Password:
  - minimo 6 caracteres
- Resposta:
  - 200 com usuario atualizado (sem password)
- Erros:
  - 400 validacao
  - 403 autorizacao
  - 404 usuario alvo nao encontrado
  - 500 erro interno
- Documentacao:
  - atualizar openapi.yaml na mesma entrega

## Escopo por camada

### 1) HTTP Router

Arquivo alvo:
- app/internal/infrastructure/http/router/router.go

Subtarefas:
1. Registrar PATCH /api/v1/users/{id}.
2. Proteger com middleware de role users:write (considerando wildcard ja suportado no middleware).

Criterio de pronto:
- rota registrada e protegida por role correta.

### 2) HTTP Handler

Arquivo alvo:
- app/internal/infrastructure/http/handler/user_handler.go

Subtarefas:
1. Criar DTO de request para PATCH com campos opcionais name, password, roles.
2. Implementar handler PatchUser:
   - ler id da rota
   - ler claims do contexto (userId, roles)
   - validar payload (ao menos 1 campo informado)
   - aplicar regras owner vs nao-owner
   - chamar use case
   - mapear erros para 400/403/404/500
3. Criar DTO de response sem password.

Criterio de pronto:
- handler retorna 200 com payload correto e mapeamento de erros consistente.

### 3) Dominio (use case + contratos)

Arquivos alvo:
- app/internal/domain/usecase (novo contrato para patch de usuario)
- app/internal/domain/repository/auth_user_repository.go

Subtarefas:
1. Criar interface de use case para patch de usuario com input explicito.
2. Estender AuthUserRepository com operacoes necessarias:
   - FindByID
   - Update (ou UpdatePartial) para name/password/roles
3. Definir erros de dominio reaproveitaveis para validacao/nao encontrado.

Criterio de pronto:
- contratos compilam e isolam regra de negocio fora do handler.

### 4) Application (regras de negocio)

Arquivo alvo:
- app/internal/application/usecase (novo arquivo patch user)

Subtarefas:
1. Implementar use case PatchUser:
   - carregar usuario alvo por id (Postgres)
   - validar alteracoes permitidas por ownership
   - validar roles contra lista fixa
   - validar password minima
   - aplicar hash quando password vier
   - persistir alteracoes no Postgres
2. Sincronizacao com Neo4j:
   - se name mudar, atualizar perfil no Neo4j via UserRepository.UpdateProfile
   - nao propagar roles/password para Neo4j

Criterio de pronto:
- regras de negocio centralizadas e sem duplicacao no handler.

### 5) Infra Postgres

Arquivo alvo:
- app/internal/infrastructure/postgres/user_repository.go

Subtarefas:
1. Implementar FindByID no repositorio de auth.
2. Implementar Update/UpdatePartial em users (Postgres).
3. Garantir que update parcial preserve campos nao enviados.

Criterio de pronto:
- repositorio suporta leitura por id e update parcial com transacao segura quando necessario.

### 6) Composicao/DI

Arquivo alvo:
- app/cmd/api/main.go

Subtarefas:
1. Registrar novo use case de patch no grafo de DI.
2. Injetar dependencia no UserHandler.
3. Ajustar construtor NewUserHandler e chamadas existentes de teste.

Criterio de pronto:
- aplicacao sobe sem erro de wiring.

### 7) Testes

Arquivos alvo:
- app/internal/application/usecase (novo *_test.go)
- app/internal/infrastructure/http/handler (novo teste de integracao)

Subtarefas:
1. Unitarios do use case:
   - owner altera name
   - owner altera password (hash aplicado)
   - owner altera roles
   - nao-owner altera apenas roles
   - nao-owner tentando alterar name/password => erro de autorizacao
   - roles invalidas => 400
   - password < 6 => 400
   - alvo inexistente => 404
2. Integracao HTTP:
   - 200 owner com patch valido
   - 200 nao-owner atualizando somente roles
   - 403 nao-owner com name/password
   - 400 body vazio/roles invalidas/password curta
   - 404 id inexistente

Criterio de pronto:
- cobertura dos cenarios de regra e autorizacao; go test ./... verde.

### 8) OpenAPI

Arquivo alvo:
- openapi.yaml

Subtarefas:
1. Adicionar operacao PATCH /api/v1/users/{id}.
2. Definir schema de request parcial.
3. Definir schema de response sem password.
4. Documentar respostas 200/400/403/404/500.
5. Documentar regra de ownership no texto da operacao.

Criterio de pronto:
- contrato OpenAPI alinhado ao comportamento implementado.

## Ordem de implementacao sugerida

1. Contratos de dominio/repositorio
2. Use case de patch
3. Repositorio Postgres (find/update)
4. Handler + router
5. DI
6. Testes unitarios
7. Testes de integracao
8. OpenAPI
9. Validacao final (go test ./...)

## Checklist de pronto

- [ ] PATCH /api/v1/users/{id} implementado
- [ ] Regra owner vs nao-owner aplicada
- [ ] users:write/* exigido para acesso
- [ ] roles validadas contra lista fixa
- [ ] password minima (>= 6) aplicada
- [ ] name sincronizado no Neo4j quando alterado
- [ ] response sem password
- [ ] openapi.yaml atualizado
- [ ] go test ./... verde

## Evidencias esperadas

- Lista de arquivos alterados por camada
- Resultado de go test ./...
- Exemplos HTTP de sucesso e erro
- Trecho do OpenAPI com nova operacao PATCH
