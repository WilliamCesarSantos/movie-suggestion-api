# EXE-03 - Remover endpoint GET /movies

Status: TODO
Prioridade: MEDIA
Dependencias: EXE-01, EXE-02
Bloqueia: EXE-04, EXE-05

## Objetivo

Remover GET /api/v1/movies e eliminar codigo associado que ficar sem uso, preservando GET /api/v1/movies/{id}.

## Entradas do Clarify

- Remover tudo que ficar sem uso relacionado ao endpoint de listagem.

## Subtarefas

1. Router
- Remover rota GET /api/v1/movies.

2. Handler/use case/repositorio
- Identificar e remover metodos de listagem nao mais utilizados.
- Preservar fluxo de detalhe por id.

3. Testes
- Atualizar ou remover testes ligados a GET /movies.
- Adicionar teste de nao disponibilidade (404/405 conforme roteamento).
- Revalidar GET /movies/{id}.

4. Documentacao local
- Remover exemplos locais que citam GET /movies (README, se aplicavel).

## Checklist de pronto

- [x] GET /api/v1/movies removido
- [x] GET /api/v1/movies/{id} mantido funcional
- [x] Sem referencias residuais no codigo runtime
- [x] Testes atualizados
- [x] go test ./... verde

## Evidencias esperadas

- Diff de rotas
- Resultado de teste do endpoint removido
