# Tarefa 2.0: Criar schema analítico e repositório base

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Definir e implementar a base persistente da funcionalidade: migrations, constraints, índices, tabelas analíticas, checkpoint e repositório responsável por `UPSERT` idempotente.

<skills>
### Conformidade com Skills Padrões

- `supabase-postgres-best-practices`: aplicar boas práticas de modelagem relacional, índices, constraints e `UPSERT`.
- `golang-patterns`: manter a camada de storage coesa, com interfaces claras e erros contextualizados.
</skills>

<requirements>
- Criar as tabelas `player_canonical`, `player_aliases`, `player_match_stats` e `player_stats_checkpoints`.
- Implementar constraints e índices previstos na techspec.
- Garantir rastreabilidade entre `player_match_stats` e `collector_matches`.
- Implementar repositório analítico com escrita idempotente e checkpoint persistido.
- Manter compatibilidade com reprocessamento seguro sem duplicação lógica.
</requirements>

## Subtarefas

- [ ] 2.1 Criar migrations iniciais do schema analítico.
- [ ] 2.2 Implementar modelos e contratos de persistência do repositório.
- [ ] 2.3 Implementar `UPSERT` transacional para identidades, aliases, fatos e checkpoint.
- [ ] 2.4 Validar índices, constraints e chaves naturais para reprocessamento idempotente.
- [ ] 2.5 Cobrir migrations e storage com testes PostgreSQL.

## Detalhes de Implementação

Referenciar na `techspec.md`:

- `Design de Implementação > Modelos de Dados`
- `Pontos de Integração`
- `Abordagem de Testes > Testes de Integração`
- `Considerações Técnicas > Decisões Principais`

## Critérios de Sucesso

- As migrations criam o schema analítico completo sem quebrar o schema existente.
- O repositório persiste fatos e identidades de forma idempotente.
- O checkpoint pode ser salvo e reutilizado em execuções futuras.
- Reprocessar o mesmo lote não duplica `player_match_stats`.

## Testes da Tarefa

- [ ] Testes de unidade para contratos e validações da camada de storage.
- [ ] Testes de integração PostgreSQL para migrations, `UPSERT`, unicidade e checkpoint.
- [ ] Testes de integração para reprocessamento do mesmo lote sem duplicação.
- [ ] Testes E2E (se aplicável): não se aplica nesta tarefa.

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `tasks/prd-consolidacao-player-stats/prd.md`
- `tasks/prd-consolidacao-player-stats/techspec.md`
- `jobs/player-stats/internal/storage`
- `jobs/player-stats/internal/model`
- `jobs/player-stats/internal/storage/migrations`
- `jobs/collector/internal/storage/migrations/001_create_collector_matches.sql`
