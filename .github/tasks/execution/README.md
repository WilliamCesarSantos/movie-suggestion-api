# Plano de Execucao - Backlog Refinado

Este diretorio contem a quebra do arquivo de refinamento em tarefas executaveis para planejamento posterior.

## Ordem de execucao sugerida

1. EXE-01 - Listar usuarios com regra por perfil
2. EXE-02 - Paginacao de recommendations (cursor)
3. EXE-03 - Remocao de GET /movies
4. EXE-04 - Atualizacao de openapi.yaml
5. EXE-06 - Revisao de logs + correlationId fim a fim
6. EXE-05 - Limpeza de arquivos/funcoes sem uso
7. EXE-08 - Renomear recomendacoes para GET /movies

## Arquivos

- EXE-01: task-01-list-users.md
- EXE-02: task-02-recommendations-pagination.md
- EXE-03: task-03-remove-get-movies.md
- EXE-04: task-04-openapi-update.md
- EXE-05: task-05-unused-code-cleanup.md
- EXE-06: task-06-logging-correlation-pod.md
- EXE-07: task-07-users-patch-update.md
- EXE-08: task-08-rename-recommendations-to-movies.md

## Regras gerais

- Cada tarefa deve ser executada em branch propria.
- Toda tarefa deve terminar com testes passando (go test ./...).
- Quando aplicavel, validar tambem contrato HTTP com testes de integracao.
- Nao executar limpeza estrutural (EXE-05) antes das mudancas funcionais e de observabilidade.
