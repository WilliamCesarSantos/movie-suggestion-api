# EXE-01 - Endpoint de listagem de usuarios com regra por perfil

Status: TODO
Prioridade: ALTA
Dependencias: nenhuma
Bloqueia: EXE-03, EXE-04, EXE-05

## Objetivo

Implementar GET /api/v1/users com autorizacao por role e regra de visibilidade:
- users:read apenas -> retorna lista com 1 item (o proprio usuario)
- users:write ou wildcard -> pode listar multiplos usuarios

## Entradas do Clarify

- Endpoint novo: GET /api/v1/users
- Filtros opcionais: email exato, name contains, page/pageSize

## Subtarefas

1. Dominio e contratos
- Adicionar interface/assinatura de listagem de usuarios em use case e repositorio.
- Definir DTO de filtros e DTO de resposta paginada de usuarios.

2. Repositorios
- Implementar consulta de listagem no repositorio fonte de verdade para usuarios autenticaveis.
- Aplicar filtros opcionais (email, name) e pagina (page/pageSize).

3. Application use case
- Implementar regra de autorizacao funcional:
  - somente users:read => forcando filtro pelo usuario do token
  - users:write/wildcard => sem restricao de identidade

4. HTTP
- Criar handler GET /api/v1/users.
- Ler claims do contexto (email, roles).
- Validar parametros de pagina/filtro e retornar 400 em invalido.

5. Router e middleware
- Registrar rota com middleware de role apropriado.
- Garantir compatibilidade com politica atual de RBAC.

6. Testes
- Unitarios do use case para regra de visibilidade.
- Integracao HTTP para:
  - users:read retornando so o proprio usuario
  - users:write retornando multiplos
  - sem role retornando 403

## Checklist de pronto

- [ ] Endpoint GET /api/v1/users implementado
- [ ] Regra users:read vs users:write aplicada
- [ ] Filtros opcionais email/name + page/pageSize implementados
- [ ] Erros de validacao retornam 400
- [ ] Testes unitarios e integracao cobrindo cenarios principais
- [ ] go test ./... verde

## Evidencias esperadas

- Lista de arquivos alterados
- Resultado de testes
- Exemplo de request/response para ambos os perfis
