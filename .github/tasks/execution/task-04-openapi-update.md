# EXE-04 - Atualizar openapi.yaml

Status: TODO
Prioridade: ALTA
Dependencias: EXE-01, EXE-02, EXE-03
Bloqueia: EXE-05

## Objetivo

Atualizar openapi.yaml para refletir exatamente o comportamento implementado.

## Entradas do Clarify

- Atualizar somente openapi.yaml.

## Subtarefas

1. Usuarios
- Adicionar operacao GET /api/v1/users.
- Documentar regra de visibilidade por perfil (users:read vs users:write).
- Documentar filtros opcionais (email, name, page, pageSize).

2. Suggestions
- Atualizar GET /api/v1/suggestions para cursor-based.
- Incluir campos de resposta: data, nextCursor, prevCursor, hasNext, hasPrev, limit, count, total.
- Documentar erros de validacao de cursor/limit.

3. Movies
- Remover operacao GET /api/v1/movies.
- Manter GET /api/v1/movies/{id} conforme implementacao.

4. Seguranca e exemplos
- Revisar security schemes e requirements por endpoint.
- Atualizar exemplos de request/response para contratos novos.

## Checklist de pronto

- [ ] openapi.yaml atualizado para todos endpoints alterados
- [ ] Sem endpoints obsoletos
- [ ] Parametros e respostas alinhados com implementacao
- [ ] Revisao manual de consistencia concluida

## Evidencias esperadas

- Diff de openapi.yaml
- Tabela resumida de endpoints alterados
