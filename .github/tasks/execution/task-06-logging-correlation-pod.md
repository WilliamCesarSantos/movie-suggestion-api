# EXE-06 - Padronizacao de logs + correlationId + pod

Status: TODO
Prioridade: ALTA
Dependencias: EXE-01, EXE-02 recomendadas
Bloqueia: EXE-05

## Objetivo

Padronizar logs da aplicacao e garantir rastreabilidade ponta a ponta, incluindo propagacao de correlationId no SQS e identificacao do pod por hostname.

## Entradas do Clarify

- Campos minimos: timestamp, level, message, correlationId, pod, route/usecase, userEmail
- Pod: hostname do container
- SQS: correlationId via MessageAttributes apenas
- Se nao vier correlationId, gerar e propagar

## Subtarefas

1. Contrato de logging
- Definir helper/utilitario para campos obrigatorios.
- Garantir adicao do campo pod em todos fluxos.

2. Fluxo HTTP
- Garantir correlationId em entrada e saida.
- Garantir route/usecase e userEmail quando disponivel.

3. SQS produtor
- Incluir correlationId em MessageAttributes.
- Garantir que correlationId da requisicao seja reaproveitado.

4. SQS consumidor
- Ler correlationId de MessageAttributes.
- Reconstruir contexto de log usando o mesmo correlationId.

5. Testes
- Integracao para validar propagacao HTTP -> SQS -> consumidor.
- Validar logs com campos obrigatorios em pontos criticos.

## Checklist de pronto

- [x] Campos obrigatorios presentes nos logs relevantes
- [x] Campo pod presente via hostname
- [x] CorrelationId propagado em MessageAttributes no produtor
- [x] CorrelationId recuperado e usado no consumidor
- [x] Testes de integracao comprovando fluxo ponta a ponta
- [x] go test ./... verde

## Evidencias esperadas

- Exemplo de logs correlacionados do mesmo fluxo
- Evidencia de message attribute no envio SQS
- Resultado dos testes

## Evidencias (2026-11-18) - revisao de campos obrigatorios

Ajustes:
- `userEmail` injetado no logger da requisicao pelo AuthMiddleware (`UpdateContext`), presente em todos os logs autenticados, inclusive no access log. Removidos campos `userEmail`/`email` redundantes no fluxo de recomendacoes (zerolog nao deduplica chaves). `email` permanece apenas quando se refere ao usuario alvo (login, cadastro, repositorios).
- Access log (`request completed`) com `logger=http.access` e `route` (pattern do chi, ex.: `/api/v1/movies/{id}`).
- Logs do AuthMiddleware com `logger=http.auth`; logs de bootstrap com `logger=bootstrap`.
- Erros do SDK OpenTelemetry roteados para o zerolog (`logger=otel`) em vez de texto puro no stdout.
- Consumidor SQS: sem `userEmail` por decisao (apenas `correlationId` via MessageAttributes); o fluxo e rastreado pelo `correlationId`.

Validacao com a stack local (no-observability), fluxos login, import, recomendacoes, detalhe, watched, users e 401:
- 122/122 linhas JSON, 0 chaves duplicadas, 0 campos obrigatorios ausentes.
- Fluxo `X-Correlation-ID: e2e-import-1`: 78 logs (HTTP -> use case -> SQS publisher -> consumidor -> Neo4j) com o mesmo `correlationId`.

Exemplo (mesmo fluxo):
```json
{"level":"info","pod":"6c98ffb47ffc","correlationId":"e2e-import-1","method":"POST","path":"/api/v1/movies-import","userEmail":"william_cesar_santos@hotmail.com","logger":"http.access","route":"/api/v1/movies-import","status":202,"durationMs":0.408,"time":"2026-11-18T03:19:06Z","message":"request completed"}
{"level":"info","pod":"6c98ffb47ffc","correlationId":"e2e-import-1","method":"POST","path":"/api/v1/movies-import","userEmail":"william_cesar_santos@hotmail.com","logger":"sqs.publisher","imdbId":"tt0078748","time":"2026-11-18T03:19:07Z","message":"import message published"}
```

Testes:
- `app/internal/infrastructure/http/middleware/logging_test.go` valida campos obrigatorios, `route` e ausencia de chaves duplicadas em requisicao autenticada.
- `go test ./...` verde (incluindo `-race` em `http/...`).
