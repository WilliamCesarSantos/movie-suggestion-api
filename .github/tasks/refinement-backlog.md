# Backlog de Refinamento - Movie Suggestion API

Objetivo: definir seis tarefas para refinamento antes da execucao, com criterios de aceite claros e pontos de atencao.

## Resultado do Clarify (consolidado)

Status do processo: CONCLUIDO

Definicoes consolidadas:
- Tarefa 1:
  - Endpoint novo: GET /api/v1/users.
  - Se role apenas users:read: retornar lista com 1 item (o proprio usuario).
  - Se users:write (ou wildcard): pode listar multiplos usuarios.
  - Filtros desta entrega: email exato, name (contains), page/pageSize.
  - Filtros sao opcionais.
- Tarefa 2:
  - Paginacao de /recommendations: cursor-based.
  - Cursor: opaco assinado.
  - Metadados de resposta: data, nextCursor, prevCursor, hasNext, hasPrev, limit, count, total.
  - Ordenacao: manter ranking atual e usar id asc apenas para desempate.
- Tarefa 3:
  - Remover GET /api/v1/movies e remover tudo que ficar sem uso relacionado a essa listagem.
- Tarefa 4:
  - Atualizar somente openapi.yaml.
- Tarefa 5:
  - Remover codigo sem uso (incluindo o usado apenas por testes).
  - Corrigir testes que quebrarem por conta da limpeza.
  - Nao remover artefatos de desenvolvimento (scripts locais, seeds, demo).
  - Remover testes obsoletos apos a limpeza.
- Tarefa 6:
  - Campos minimos de log: timestamp, level, message, correlationId, pod, route/usecase, userEmail.
  - Identificacao de pod: hostname do container.
  - Propagacao de correlationId em SQS: MessageAttributes apenas.
  - Se requisicao sem correlationId: gerar e propagar.

Pendencias de clarify:
- Nenhuma pendencia aberta neste momento.

## Quebra para execucao posterior

Arquivos de execucao gerados em .github/tasks/execution:
- README.md
- task-01-list-users.md
- task-02-recommendations-pagination.md
- task-03-remove-get-movies.md
- task-04-openapi-update.md
- task-05-unused-code-cleanup.md
- task-06-logging-correlation-pod.md

## Tarefa 1 - Endpoint para listar usuarios com regra por perfil

Status: PENDENTE
Prioridade: ALTA

Descricao:
- Criar endpoint para listar usuarios.
- Permissoes aceitas: users:read e users:write.
- Regra obrigatoria:
  - quando o usuario logado tiver apenas users:read, retornar somente o proprio usuario.
  - quando tiver users:write (ou role wildcard), permitir retorno de multiplos usuarios.

Escopo tecnico inicial:
- Ajustar contrato de caso de uso para listagem.
- Criar endpoint HTTP e wiring no router.
- Aplicar regra de autorizacao por role e por identidade do token.
- Validar obtencao de email/identidade do token para filtrar corretamente.

Criterios de aceite:
- Usuario com apenas users:read recebe apenas 1 registro (ele mesmo).
- Usuario com users:write consegue listar mais de um usuario.
- Sem permissao users:read/users:write retorna 403.
- Testes unitarios e de integracao cobrindo os cenarios.

## Tarefa 2 - Paginacao no endpoint de recommendations

Status: PENDENTE
Prioridade: ALTA

Descricao:
- Atualizar GET /api/v1/recommendations para suportar paginacao.

Escopo tecnico inicial:
- Definir contrato de paginacao (page, pageSize) ou cursor.
- Ajustar handler, caso de uso e repositorio para pagina.
- Definir limites maximos para pageSize.
- Retornar metadados de paginacao no payload (exemplo: page, pageSize, total, hasNext).

Criterios de aceite:
- Endpoint retorna pagina solicitada sem quebrar autorizacao atual.
- Valores invalidos de pagina retornam 400 com mensagem clara.
- Paginacao respeita limite maximo configuravel.
- Testes para primeira pagina, pagina intermediaria e pagina vazia.

## Tarefa 3 - Remover endpoint GET /movies

Status: PENDENTE
Prioridade: MEDIA

Descricao:
- Remover endpoint GET /api/v1/movies que retorna lista de filmes.

Escopo tecnico inicial:
- Remover rota no router.
- Remover handler/caso de uso nao mais necessarios para listagem.
- Atualizar testes e exemplos de uso.

Criterios de aceite:
- GET /api/v1/movies nao deve mais estar disponivel.
- GET /api/v1/movies/{id} permanece funcional.
- Sem referencias residuais ao endpoint removido em codigo e docs.

## Tarefa 4 - Atualizar documentacao OpenAPI

Status: PENDENTE
Prioridade: ALTA

Descricao:
- Atualizar o arquivo OpenAPI para refletir as mudancas funcionais.

Escopo tecnico inicial:
- Ajustar especificacao em openapi.yaml (caminhos, schemas, seguranca, respostas).
- Documentar endpoint de listagem de usuarios e regra por perfil.
- Documentar paginacao de recommendations.
- Remover operacao de GET /movies.

Criterios de aceite:
- openapi.yaml representa exatamente os endpoints atuais.
- Contratos de request/response e codigos HTTP estao alinhados com implementacao.
- Exemplos de payload e parametros atualizados.

## Tarefa 5 - Analise de uso e remocao de arquivos/funcoes nao utilizados

Status: PENDENTE
Prioridade: ALTA

Descricao:
- Fazer analise completa para identificar arquivos e funcoes sem uso em runtime.
- Arquivos/funcoes usados apenas por testes devem ser considerados nao utilizados para este objetivo.
- Remover o que for comprovadamente nao utilizado.

Escopo tecnico inicial:
- Gerar inventario de arquivos/funcoes e mapa de referencias.
- Classificar: usado em runtime, usado so em teste, sem uso.
- Remover itens sem uso com seguranca.
- Atualizar imports, wiring e testes impactados.

Criterios de aceite:
- Lista de remocoes com justificativa objetiva por item.
- Build da aplicacao sem erros apos remocoes.
- Testes principais e smoke checks de rotas criticas sem regressao.
- Sem codigo morto residual identificado na rodada final.

Observacao de risco:
- Essa tarefa exige refinamento cuidadoso para evitar falso positivo em uso dinamico (injecao, reflexao, inicializacao por side effects).

## Tarefa 6 - Revisao e padronizacao de logs + correlationId ponta a ponta

Status: PENDENTE
Prioridade: ALTA

Descricao:
- Revisar logs de todos os fluxos para garantir rastreabilidade e padrao minimo.
- Garantir publicacao de correlationId em logs de entrada, processamento e saida.
- Em publicacoes SQS, propagar correlationId da requisicao para a mensagem.
- No consumidor SQS, recuperar e usar o mesmo correlationId durante o processamento.
- Incluir identificacao do pod que gerou o log.

Escopo tecnico inicial:
- Definir contrato de logging (campos obrigatorios).
- Aplicar logs nos fluxos HTTP, casos de uso criticos e integracoes (DB, SQS, OMDB).
- Evoluir payload/metadata da mensagem SQS para levar correlationId.
- Ajustar consumidor para restaurar contexto de log.
- Incluir campo pod (exemplo: hostname ou env POD_NAME).

Criterios de aceite:
- Todos os logs relevantes incluem correlationId.
- CorrelationId da requisicao aparece tambem no produtor e consumidor SQS para o mesmo fluxo.
- Logs incluem identificador do pod.
- Cobertura por testes de integracao para propagacao do correlationId em SQS.

## Ordem sugerida para execucao apos refinamento

1. Tarefa 1
2. Tarefa 2
3. Tarefa 3
4. Tarefa 4
5. Tarefa 6
6. Tarefa 5

Justificativa:
- Primeiro mudancas funcionais e contrato API.
- Depois observabilidade e rastreabilidade.
- Por fim limpeza estrutural (maior risco) com base no estado final do sistema.
