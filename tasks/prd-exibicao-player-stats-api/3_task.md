# Tarefa 3.0: Implementar consulta SQL agregada e acesso PostgreSQL do ranking

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Criar a camada de storage responsável por consultar a base analítica existente e retornar linhas agregadas de ranking com filtros combináveis, exclusão de partidas inelegíveis, mínimo de 10 partidas e ordenação determinística.

<skills>
### Conformidade com Skills Padrões

- `supabase-postgres-best-practices`: aplicar boas práticas de agregação, filtros, índices e análise de custo da query principal.
- `golang-patterns`: manter a camada de storage coesa, com contratos claros e erros contextualizados.
</skills>

<requirements>
- Ler `player_match_stats` e `player_canonical` como fonte do ranking.
- Filtrar sempre `excluded_from_analytics = false`.
- Aplicar filtros por modo, mapa, servidor e período de forma combinável.
- Garantir `HAVING count(*) >= 10` para elegibilidade no ranking.
- Implementar ordenação por campos permitidos com desempate determinístico.
</requirements>

## Subtarefas

- [ ] 3.1 Definir o contrato de repositório para listagem do ranking.
- [ ] 3.2 Implementar a consulta SQL agregada com joins, filtros e agregações necessárias.
- [ ] 3.3 Implementar paginação por `limit` e `offset` e cálculo de `has_next`.
- [ ] 3.4 Validar a estratégia de ordenação parametrizada com whitelist segura.
- [ ] 3.5 Cobrir a query principal com testes PostgreSQL e análise de comportamento.

## Detalhes de Implementação

Referenciar na `techspec.md`:

- `Design de Implementação > Modelos de Dados`
- `Design de Implementação > Endpoints de API`
- `Pontos de Integração`
- `Abordagem de Testes > Testes de Integração`
- `Considerações Técnicas > Riscos Conhecidos`

## Critérios de Sucesso

- O repositório retorna ranking agregado correto para os filtros suportados.
- Partidas com `excluded_from_analytics = true` nunca entram na consulta pública.
- O mínimo de 10 partidas é aplicado de forma consistente.
- Empates respeitam o desempate determinístico definido na spec.

## Testes da Tarefa

- [ ] Testes de unidade para helpers de montagem segura de ordenação, se existirem.
- [ ] Testes de integração PostgreSQL para ranking sem filtros.
- [ ] Testes de integração PostgreSQL para filtros combinados, mínimo de partidas e exclusão analítica.
- [ ] Testes de integração PostgreSQL para paginação e ordenação determinística.
- [ ] Testes E2E (se aplicável): não se aplica nesta tarefa.

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `tasks/prd-exibicao-player-stats-api/prd.md`
- `tasks/prd-exibicao-player-stats-api/techspec.md`
- `services/player-stats-api/internal/storage`
- `services/player-stats-api/internal/model`
- `jobs/player-stats/internal/storage/migrations/001_create_player_stats_analytics.sql`
- `jobs/player-stats/internal/model/analytics.go`
