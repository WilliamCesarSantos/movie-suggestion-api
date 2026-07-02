# EXE-02 - Paginacao cursor-based em /suggestions

Status: TODO
Prioridade: ALTA
Dependencias: EXE-01 recomendada
Bloqueia: EXE-03, EXE-04, EXE-05

## Objetivo

Atualizar GET /api/v1/suggestions para paginacao cursor-based com cursor opaco assinado, preservando ranking atual e usando id asc apenas como desempate.

## Entradas do Clarify

- Modelo: cursor-based
- Cursor: opaco assinado
- Metadados: data, nextCursor, prevCursor, hasNext, hasPrev, limit, count, total
- Ordenacao: ranking atual + id asc como desempate

## Subtarefas

1. Contrato de API interna
- Definir estrutura de entrada (cursor, limit).
- Definir estrutura de saida com metadados exigidos.

2. Token de cursor
- Implementar encode/decode opaco assinado.
- Definir tratamento para cursor invalido (400).

3. Query e repositorio
- Ajustar consultas para pagina por cursor.
- Garantir ordenacao estavel conforme regra de negocio.
- Implementar busca de pagina anterior (prevCursor) sem inconsistencias.

4. Use case
- Integrar logica de paginacao ao fluxo de sugestao atual.
- Manter fallback de algoritmo existente sem quebrar a pagina.

5. Handler HTTP
- Validar limit, cursor e parametros.
- Retornar payload paginado com metadados acordados.

6. Testes
- Unitarios de cursor (encode/decode e assinatura).
- Integracao para primeira pagina, pagina seguinte, pagina vazia e cursor invalido.

## Checklist de pronto

- [ ] /suggestions suporta cursor + limit
- [ ] Cursor opaco assinado implementado
- [ ] Metadados completos no response
- [ ] Ordenacao estavel validada
- [ ] Testes cobrindo cenarios principais
- [ ] go test ./... verde

## Evidencias esperadas

- Exemplo de duas paginas consecutivas
- Comprovacao de hasNext/hasPrev
- Resultado dos testes
