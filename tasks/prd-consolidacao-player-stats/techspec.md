# Template de Especificação Técnica

## Resumo Executivo

A consolidacao de player stats sera implementada como um job Go separado do `match-stats-collector`, executado de forma independente e tendo `collector_matches` como unica fonte de verdade. O job fara uma execucao batch: carregar configuracao e conexao, aplicar migracoes proprias, selecionar partidas elegiveis, extrair jogadores do `stats_payload`, normalizar o modo em um conjunto canonico, resolver identidade inicial por `login` e alias observado, e persistir tabelas analiticas idempotentes em PostgreSQL.

As decisoes centrais sao: manter a responsabilidade do coletor atual intacta; modelar a camada analitica em tabelas relacionais separadas para identidades, aliases e fatos por jogador/partida; usar `UPSERT` com chaves naturais derivadas da partida e do jogador para permitir reprocessamento seguro; e marcar partidas com bots como excluidas do consumo padrao por coluna dedicada, sem descartar o historico bruto. A observabilidade seguira o padrao existente do projeto com logs estruturados e metricas Prometheus.

## Arquitetura do Sistema

### Visão Geral dos Componentes

- `jobs/player-stats/cmd/player-stats`: entrypoint do novo binario batch.
- `jobs/player-stats/internal/config`: carrega env e parametros do job, reutilizando o padrao do collector.
- `jobs/player-stats/internal/bootstrap`: abre pool PostgreSQL, aplica migracoes, inicializa metrics, logger e service.
- `jobs/player-stats/internal/source`: le partidas de `collector_matches` em lotes ordenados por `played_at` e `id`.
- `jobs/player-stats/internal/normalize`: normaliza modo canonico, flags analiticas e extracao das metricas por jogador.
- `jobs/player-stats/internal/identity`: resolve `canonical_player` e `alias` usando primeiro `login` conhecido e, na ausencia dele, o nome observado.
- `jobs/player-stats/internal/storage`: persiste entidades analiticas e watermark de processamento com `UPSERT`.
- `jobs/player-stats/internal/metrics`: expoe metricas Prometheus no mesmo formato operacional do collector.

Relacionamentos principais:

- `cmd/player-stats` depende de `bootstrap`.
- `bootstrap` conecta `config`, `storage`, `metrics` e `service`.
- `service` usa `source` para leitura, `normalize` para transformacao, `identity` para resolucao de jogadores e `storage` para escrita.
- O schema analitico depende de `collector_matches`, mas nao altera sua semantica.

Fluxo de dados:

1. O job inicia, carrega configuracao e aplica migracoes analiticas.
2. Busca partidas em `collector_matches` ainda nao consolidadas ou candidatas a reprocessamento.
3. Para cada partida com `stats_payload` valido, extrai jogadores, modo bruto, `dm`, mapa, time e metricas.
4. O normalizador define `normalized_mode`: `dmm4` quando `dm=4`; caso contrario classifica por quantidade de jogadores ativos em `1on1`, `2on2`, `3on3` ou `4on4`.
5. O resolvedor de identidade procura jogador existente por `login`; se nao encontrar, procura alias identico; se ainda nao existir, cria novo jogador canonico.
6. O storage grava dimensoes de identidade e fatos por jogador/partida na mesma transacao.
7. O job atualiza o watermark e publica logs e metricas do ciclo.

## Design de Implementação

### Interfaces Principais

```go
type MatchSource interface {
    ListMatchesForConsolidation(ctx context.Context, cursor Cursor, limit int) ([]model.SourceMatch, Cursor, error)
}

type IdentityResolver interface {
    ResolvePlayer(ctx context.Context, input ResolvePlayerInput) (model.PlayerIdentity, error)
}

type AnalyticsRepository interface {
    UpsertBatch(ctx context.Context, batch ConsolidationBatch) (BatchResult, error)
    SaveCheckpoint(ctx context.Context, checkpoint Checkpoint) error
}
```

```go
type ConsolidationService interface {
    RunOnce(ctx context.Context) error
    ConsolidateMatch(ctx context.Context, match model.SourceMatch) (ConsolidationBatch, error)
}
```

### Modelos de Dados

Entidades principais:

- `player_canonical`
  - `id uuid primary key`
  - `primary_login text null`
  - `display_name text not null`
  - `created_at timestamptz`
  - `updated_at timestamptz`
  - unique parcial em `primary_login` quando nao nulo

- `player_aliases`
  - `id bigserial primary key`
  - `player_id uuid not null references player_canonical(id)`
  - `alias_name text not null`
  - `login text null`
  - `first_seen_at timestamptz not null`
  - `last_seen_at timestamptz not null`
  - unique em `(alias_name, coalesce(login, ''))`

- `player_match_stats`
  - `id bigserial primary key`
  - `collector_match_id bigint not null references collector_matches(id)`
  - `player_id uuid not null references player_canonical(id)`
  - `server_key text not null`
  - `demo_name text not null`
  - `observed_name text not null`
  - `observed_login text null`
  - `team text null`
  - `map_name text not null`
  - `raw_mode text null`
  - `normalized_mode text not null`
  - `played_at timestamptz not null`
  - `has_bots boolean not null`
  - `excluded_from_analytics boolean not null`
  - metricas base: `frags`, `deaths`, `kills`, `team_kills`, `suicides`, `damage_taken`, `damage_given`, `spree_max`, `spree_quad`, `rl_hits`, `rl_kills`, `lg_attacks`, `lg_hits`, `ga`, `ra`, `ya`, `health_100`, `ping`
  - metricas derivadas: `efficiency numeric(5,2)`, `lg_accuracy numeric(5,2)`
  - `stats_snapshot jsonb not null`
  - `consolidated_at timestamptz not null`
  - unique em `(collector_match_id, player_id, observed_name)`

- `player_stats_checkpoints`
  - `job_name text primary key`
  - `last_collector_match_id bigint not null`
  - `updated_at timestamptz not null`

Regras de modelagem:

- `collector_match_id` preserva rastreabilidade direta com a linha bruta.
- `stats_snapshot` armazena apenas o recorte bruto do jogador extraido de `stats_payload`; nao replica o payload completo da partida.
- `excluded_from_analytics` sera igual a `has_bots`, permitindo filtro simples sem apagar fatos.
- `normalized_mode` tera `CHECK` para `1on1`, `2on2`, `3on3`, `4on4`, `dmm4`.
- `efficiency` sera calculada no job para simplificar consumo posterior; quando o denominador for zero, o valor sera `0`.
- `lg_accuracy` sera calculada como `lg_hits / lg_attacks * 100`; quando `lg_attacks=0`, o valor sera `0`.

Indices sugeridos:

- `player_match_stats(player_id, played_at desc)`
- `player_match_stats(normalized_mode, played_at desc)`
- `player_match_stats(map_name, played_at desc)`
- `player_match_stats(server_key, played_at desc)`
- `player_match_stats(excluded_from_analytics, played_at desc)`
- `player_aliases(player_id)`

### Endpoints de API

Nao se aplica. Esta entrega cria apenas tabelas analiticas em PostgreSQL.

## Pontos de Integração

- PostgreSQL existente, usando o mesmo `DATABASE_URL` e o schema onde `collector_matches` ja reside.
- Tabela `collector_matches` como fonte de leitura; o job nao depende dos endpoints `lastscores` ou `laststats` em tempo de execucao.

Tratamento de erros:

- Falha ao consolidar uma partida nao deve corromper o lote; o job deve registrar erro com `collector_match_id`, `server_key` e `demo_name`.
- `stats_payload` ausente ou invalido gera skip contabilizado, nao panic.
- Colisao de identidade por `login` deve prevalecer sobre alias por nome, conforme decisao de negocio definida.
- `UPSERT` deve ser transacional por lote para manter identidade, alias e fatos coerentes.

## Abordagem de Testes

### Testes Unidade

- Normalizacao de modo:
  - `dm=4` resulta em `dmm4`.
  - Demais partidas usam contagem de jogadores ativos.
- Extracao de metricas:
  - leitura correta de `stats`, `dmg`, `spree`, `weapons.lg.acc`, `items`.
- Calculos derivados:
  - `efficiency` e `lg_accuracy` com divisao por zero.
- Resolucao de identidade:
  - reutiliza jogador por `login`.
  - reutiliza alias conhecido.
  - cria jogador e alias novos quando necessario.
- Regras de bots:
  - fatos sao persistidos com `excluded_from_analytics=true`.

Mocks:

- mock apenas do repositorio/fonte quando testar o service; a logica de normalizacao deve permanecer pura.

### Testes de Integração

- `source + storage + PostgreSQL` com `collector_matches` preenchida por fixtures reais do projeto.
- Reprocessamento do mesmo lote sem duplicar `player_match_stats`.
- Evolucao de alias: mesma pessoa reaparecendo com mesmo `login` e nome diferente.
- Partida com bots consolidada e filtravel por `excluded_from_analytics`.
- Validacao de migracoes novas junto das migracoes ja existentes do collector.

### Testes de E2E

Nao se aplica. Nao ha frontend neste escopo.

## Sequenciamento de Desenvolvimento

### Ordem de Construção

1. Criar o esqueleto do novo job `jobs/player-stats` com `cmd`, `bootstrap`, `config`, `logging` e `metrics`, reaproveitando os padroes do collector.
2. Definir migracoes e repositorio analitico primeiro, porque a modelagem de identidade e fatos trava o restante do fluxo.
3. Implementar leitura de `collector_matches` em lotes e checkpoint de processamento.
4. Implementar normalizacao de modo e extracao de metricas por jogador.
5. Implementar resolucao de identidade por `login` e alias.
6. Integrar o service batch com transacao por lote, logs e metricas.
7. Fechar testes de integracao com PostgreSQL e validacao local em `compose.yml`.

### Dependências Técnicas

- Instancia PostgreSQL acessivel.
- `collector_matches` populada pelo job existente.
- Inclusao do novo servico opcional em `compose.yml` para validacao local.

## Monitoramento e Observabilidade

Metricas Prometheus:

- `player_stats_runs_total{status}`
- `player_stats_matches_scanned_total{result}`
- `player_stats_player_rows_upserted_total{result}`
- `player_stats_identity_resolutions_total{result}`
- `player_stats_processing_duration_seconds{stage}`
- `player_stats_skipped_matches_total{reason}`

Logs estruturados:

- inicio e fim do ciclo com quantidade de partidas lidas, consolidadas, puladas e reprocessadas
- falhas por partida com `collector_match_id`, `server_key`, `demo_name`
- criacao de jogador canonico e criacao/reuso de alias
- tamanho de lote, duracao da transacao e checkpoint final

Nao ha evidencia de dashboards Grafana existentes no repositorio atual; a spec assume apenas exposicao Prometheus compativel.

## Considerações Técnicas

### Decisões Principais

- Job separado foi escolhido para manter o coletor simples e preservar o limite de responsabilidade atual definido no PRD e no estado do projeto.
- Tabelas relacionais separadas para identidade e fatos sao preferiveis a mais `JSONB` porque o objetivo desta fase e consulta analitica eficiente por jogador, mapa, modo, servidor e periodo.
- O reaproveitamento de identidade por `login` reduz fragmentacao inicial sem introduzir heuristicas opacas; nomes diferentes com mesmo `login` convergem para o mesmo jogador canonico.
- Checkpoint por `collector_match_id` simplifica execucao incremental; ainda assim, o `UPSERT` continua obrigatorio para suportar reprocessamento seguro.

Trade-offs e alternativas rejeitadas:

- Fazer a consolidacao dentro de `jobs/collector` foi rejeitado porque acoplaria ingestao bruta e transformacao analitica no mesmo ciclo operacional.
- Resolver identidade apenas por nome foi rejeitado porque perderia a pista mais estavel ja disponivel no payload.
- Criar views materializadas sem tabelas proprias foi rejeitado porque dificultaria evolucao de identidade, auditoria e reprocessamento controlado.

### Riscos Conhecidos

- `login` pode estar ausente ou mudar entre partidas; mitigacao: preservar alias e nome observado em cada fato.
- Mudancas futuras no payload de `laststats` podem quebrar extracao de metricas; mitigacao: testes com fixtures reais e `stats_snapshot` para auditoria.
- Reprocessamento parcial pode deixar checkpoint adiantado se a transacao nao for unica; mitigacao: salvar checkpoint apenas apos commit bem-sucedido do lote.
- Contagem de jogadores ativos pode divergir de casos edge com espectadores; mitigacao: considerar apenas entradas com nome observavel e estatisticas de partida.

### Conformidade com Skills Padrões

- `golang-patterns`: aplicavel para manter packages pequenos, interfaces no boundary e erros contextualizados.
- `supabase-postgres-best-practices`: aplicavel para desenho de indices, constraints e estrategia de `UPSERT`/checkpoint.

Observacao: nao foi encontrada a pasta `@.claude/skills` mencionada no template; a avaliacao acima usa as skills disponiveis em `.agents/skills`.

### Arquivos relevantes e dependentes

- [prd.md](/home/rspassos/projects/ilha/tasks/prd-consolidacao-player-stats/prd.md)
- [techspec.md](/home/rspassos/projects/ilha/tasks/prd-consolidacao-player-stats/techspec.md)
- [main.go](/home/rspassos/projects/ilha/jobs/collector/cmd/collector/main.go)
- [bootstrap.go](/home/rspassos/projects/ilha/jobs/collector/internal/bootstrap/bootstrap.go)
- [config.go](/home/rspassos/projects/ilha/jobs/collector/internal/config/config.go)
- [service.go](/home/rspassos/projects/ilha/jobs/collector/internal/collector/service.go)
- [match.go](/home/rspassos/projects/ilha/jobs/collector/internal/model/match.go)
- [record.go](/home/rspassos/projects/ilha/jobs/collector/internal/model/record.go)
- [postgres.go](/home/rspassos/projects/ilha/jobs/collector/internal/storage/postgres.go)
- [001_create_collector_matches.sql](/home/rspassos/projects/ilha/jobs/collector/internal/storage/migrations/001_create_collector_matches.sql)
- [metrics.go](/home/rspassos/projects/ilha/jobs/collector/internal/metrics/metrics.go)
- [compose.yml](/home/rspassos/projects/ilha/compose.yml)
