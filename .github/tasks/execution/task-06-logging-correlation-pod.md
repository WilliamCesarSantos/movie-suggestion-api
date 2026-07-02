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

- [ ] Campos obrigatorios presentes nos logs relevantes
- [ ] Campo pod presente via hostname
- [ ] CorrelationId propagado em MessageAttributes no produtor
- [ ] CorrelationId recuperado e usado no consumidor
- [ ] Testes de integracao comprovando fluxo ponta a ponta
- [ ] go test ./... verde

## Evidencias esperadas

- Exemplo de logs correlacionados do mesmo fluxo
- Evidencia de message attribute no envio SQS
- Resultado dos testes
