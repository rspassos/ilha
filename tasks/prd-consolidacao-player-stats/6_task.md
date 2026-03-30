# Tarefa 6.0: Integrar o fluxo batch de consolidação

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Conectar leitura, normalização, identidade e persistência em um serviço batch `RunOnce`, com transação por lote, logs estruturados, métricas operacionais e checkpoint salvo apenas após escrita bem-sucedida.

<skills>
### Conformidade com Skills Padrões

- `golang-patterns`: orquestrar dependências por interfaces e manter o service claro e testável.
- `supabase-postgres-best-practices`: garantir atomicidade de lote e checkpoint consistente com commit.
</skills>

<requirements>
- Implementar o `ConsolidationService` com execução `RunOnce`.
- Processar partidas em lotes, consolidando registros por jogador até a persistência final.
- Garantir transação por lote para identidade, aliases, fatos e checkpoint.
- Registrar métricas e logs estruturados por ciclo, lote e falha por partida.
- Isolar erros por partida ou por lote sem corromper o progresso já confirmado.
</requirements>

## Subtarefas

- [ ] 6.1 Implementar o service de orquestração batch.
- [ ] 6.2 Integrar source, normalizer, identity resolver e repository.
- [ ] 6.3 Implementar persistência transacional por lote com atualização de checkpoint após commit.
- [ ] 6.4 Implementar logs estruturados e métricas definidas na spec.
- [ ] 6.5 Cobrir o fluxo completo com testes de unidade e integração.

## Detalhes de Implementação

Referenciar na `techspec.md`:

- `Design de Implementação > Interfaces Principais`
- `Arquitetura do Sistema > Fluxo de dados`
- `Pontos de Integração > Tratamento de erros`
- `Monitoramento e Observabilidade`
- `Considerações Técnicas > Riscos Conhecidos`

## Critérios de Sucesso

- O job executa um ciclo batch completo de consolidação ponta a ponta.
- O checkpoint só avança após persistência bem-sucedida do lote.
- Logs e métricas permitem identificar partidas lidas, consolidadas, puladas e falhas.
- Reprocessamento continua seguro mesmo após falhas parciais.

## Testes da Tarefa

- [ ] Testes de unidade do service com mocks das bordas.
- [ ] Testes de unidade para regras de checkpoint e propagação de erros.
- [ ] Testes de integração para ciclo completo de lote com PostgreSQL.
- [ ] Testes de integração para falha isolada e reexecução idempotente.
- [ ] Testes E2E (se aplicável): não se aplica nesta tarefa.

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `tasks/prd-consolidacao-player-stats/prd.md`
- `tasks/prd-consolidacao-player-stats/techspec.md`
- `jobs/player-stats/internal/service`
- `jobs/player-stats/internal/source`
- `jobs/player-stats/internal/normalize`
- `jobs/player-stats/internal/identity`
- `jobs/player-stats/internal/storage`
- `jobs/player-stats/internal/metrics`
