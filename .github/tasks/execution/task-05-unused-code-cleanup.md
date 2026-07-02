# EXE-05 - Limpeza de arquivos e funcoes sem uso

Status: TODO
Prioridade: ALTA
Dependencias: EXE-01, EXE-02, EXE-03, EXE-04, EXE-06
Bloqueia: nenhuma

## Objetivo

Identificar e remover arquivos/funcoes sem uso em runtime, incluindo casos usados apenas por testes, corrigindo impactos necessarios.

## Entradas do Clarify

- Codigo usado apenas por testes deve ser removido para este objetivo.
- Corrigir testes que quebrarem.
- Nao remover artefatos de desenvolvimento (scripts locais, seeds, demo).
- Remover testes obsoletos.

## Subtarefas

1. Inventario
- Levantar todos arquivos e funcoes do runtime.
- Mapear referencias com busca textual + referencias de simbolo.

2. Classificacao
- Runtime usado
- Somente teste
- Sem uso

3. Plano de remocao
- Montar lista com justificativa por item.
- Revisar itens de alto risco (inicializacao indireta/side effects).

4. Execucao
- Remover codigo sem uso.
- Remover testes obsoletos.
- Ajustar imports e wiring.

5. Validacao
- Build da aplicacao.
- go test ./...
- Smoke de rotas criticas.

## Checklist de pronto

- [ ] Inventario de uso documentado
- [ ] Lista de remocoes com justificativa
- [ ] Codigo/funcoes sem uso removidos
- [ ] Testes obsoletos removidos e suite ajustada
- [ ] Artefatos de desenvolvimento preservados
- [ ] go test ./... verde

## Evidencias esperadas

- Relatorio de remocao (arquivo + funcao + motivo)
- Resultado de build/testes
