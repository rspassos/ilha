# Tarefa 3.0: Implementar leitura incremental de `collector_matches`

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Criar a camada de leitura da fonte de verdade `collector_matches`, com paginação por lotes, ordenação determinística, uso de checkpoint e tratamento seguro de partidas inválidas ou não elegíveis.

<skills>
### Conformidade com Skills Padrões

- `golang-patterns`: separar claramente source, modelos de leitura e regras de paginação.
- `supabase-postgres-best-practices`: garantir consultas eficientes e compatíveis com o padrão incremental definido na spec.
</skills>

<requirements>
- Ler `collector_matches` em lotes ordenados por `played_at` e `id`.
- Suportar continuidade por checkpoint baseado em `collector_match_id`.
- Identificar partidas elegíveis à consolidação sem depender de endpoints externos.
- Tratar `stats_payload` ausente ou inválido como skip contabilizado, sem interromper o lote.
- Preservar dados necessários para rastreabilidade e posterior consolidação.
</requirements>

## Subtarefas

- [ ] 3.1 Definir o modelo de leitura de `SourceMatch` e cursor incremental.
- [ ] 3.2 Implementar consultas paginadas à tabela `collector_matches`.
- [ ] 3.3 Implementar regras de elegibilidade e skip para payload ausente ou inválido.
- [ ] 3.4 Integrar leitura com checkpoint persistido.
- [ ] 3.5 Cobrir paginação, skips e leitura real com testes automatizados.

## Detalhes de Implementação

Referenciar na `techspec.md`:

- `Arquitetura do Sistema > Fluxo de dados`
- `Design de Implementação > Interfaces Principais`
- `Pontos de Integração`
- `Abordagem de Testes > Testes de Integração`

## Critérios de Sucesso

- A fonte lista partidas de forma incremental e determinística.
- O job consegue retomar processamento a partir do último checkpoint salvo.
- Payload inválido não causa panic nem invalida o restante do lote.
- Os dados retornados pela source são suficientes para a transformação analítica das próximas tarefas.

## Testes da Tarefa

- [ ] Testes de unidade para cursor, paginação e elegibilidade.
- [ ] Testes de unidade para contagem e classificação de skips.
- [ ] Testes de integração com `collector_matches` preenchida por fixtures reais.
- [ ] Testes E2E (se aplicável): não se aplica nesta tarefa.

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `tasks/prd-consolidacao-player-stats/prd.md`
- `tasks/prd-consolidacao-player-stats/techspec.md`
- `jobs/player-stats/internal/source`
- `jobs/player-stats/internal/model`
- `jobs/player-stats/internal/storage`
- `jobs/collector/internal/storage/postgres.go`
- `jobs/collector/internal/model`
