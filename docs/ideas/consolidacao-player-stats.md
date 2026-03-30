# Plano: Consolidacao de Stats de Jogadores para Analytics

## Resumo

Criar um segundo job, separado do `jobs/collector`, responsável por ler `collector_matches`, explodir os dados por jogador e popular uma camada analítica própria para dashboard futuro.

Os payloads de `docs/responses/laststats.json` e `docs/responses/lastscores.json` mostram que:

- `laststats` é a fonte principal para stats por jogador, itens e armas.
- `lastscores` complementa identidade do jogo e ajuda em consistência de modo/participantes.
- em jogos de time, `lastscores.mode` vem como `2on2` e `laststats.mode` vem como `team`, então a normalização de modo deve acontecer no job novo, não no coletor.
- existe `ping` por jogador, `map`, `is_bot` em scores e `bot` em stats.

A solução recomendada é uma modelagem híbrida:

- tabela de dimensão de jogadores
- tabela de aliases/nomes observados
- tabela fato por jogador por partida
- opcionalmente guardar um `raw_player_payload jsonb` para reprocessamento leve sem depender só da tabela-mãe

## Mudanças de Implementação

### 1. Novo job de consolidação

Criar um novo job, por exemplo `jobs/player-stats`, com execução single-shot, idempotente, lendo partidas novas ou alteradas em `collector_matches`.

Responsabilidades:

- selecionar partidas de `collector_matches`
- ignorar da camada analítica principal os jogos com `has_bots = true`, mas preservar a flag na linha consolidada quando houver necessidade de rastreio
- extrair um registro por jogador
- normalizar modo e mapa
- popular dimensões e fato analítico com `UPSERT`

Estado de processamento:

- usar watermark por `updated_at` + `id` de `collector_matches`, salvo em tabela própria de controle do job
- permitir reprocessamento completo por flag futura, sem depender disso agora

### 2. Modelo analítico proposto

#### Tabela `players`

Finalidade: entidade canônica do jogador.

Campos mínimos:

- `id`
- `canonical_name`
- `created_at`
- `updated_at`

Regra:

- não tentar resolver identidade real agora
- criar um player canônico automaticamente quando surgir um nome ainda não conhecido

#### Tabela `player_aliases`

Finalidade: nomes observados que poderão ser vinculados depois no painel administrativo.

Campos mínimos:

- `id`
- `player_id`
- `alias_name`
- `first_seen_at`
- `last_seen_at`
- `source`
- `is_primary`

Regras:

- unicidade por `alias_name`
- na ingestão inicial, cada nome observado cria alias e player 1:1
- futura consolidação administrativa poderá mover aliases entre players sem alterar o histórico bruto de partidas

#### Tabela `player_match_stats`

Finalidade: fato analítico por jogador por partida.

Chave recomendada:

- unique `(collector_match_id, player_alias_id)`

Campos mínimos:

- `id`
- `collector_match_id`
- `server_key`
- `demo_name`
- `played_at`
- `player_id`
- `player_alias_id`
- `player_name_observed`
- `team_name`
- `mode`
- `map_name`
- `ping`
- `has_bots`
- `excluded_from_analytics`
- `frags`
- `deaths`
- `kills`
- `team_kills`
- `suicides`
- `efficiency`
- `damage_taken`
- `damage_given`
- `damage_self`
- `damage_team`
- `taken_to_die`
- `spree_max`
- `spree_quad`
- `rl_hits`
- `rl_kills`
- `lg_attacks`
- `lg_hits`
- `lg_hit_percentage`
- `ga_taken`
- `ra_taken`
- `ya_taken`
- `health_100_taken`
- `xfer_rl`
- `xfer_lg`
- `raw_player_payload jsonb`
- `ingested_at`
- `updated_at`

Defaults de cálculo:

- `efficiency = kills / (kills + deaths) * 100`, com `0` quando denominador for `0`
- `lg_hit_percentage = lg_hits / lg_attacks * 100`, com `0` quando denominador for `0`

Índices recomendados:

- `(player_id, played_at desc)`
- `(mode, played_at desc)`
- `(map_name, played_at desc)`
- `(excluded_from_analytics, played_at desc)`
- `(server_key, played_at desc)`

### 3. Normalização de modo

Criar regra explícita de derivação no job novo:

1. Os únicos modos persistidos na camada analítica devem ser `1on1`, `2on2`, `3on3`, `4on4` e `dmm4`.
2. Se `laststats.dm = 4`, gravar `mode = dmm4`.
3. Caso contrário, derivar pelo número de jogadores ativos na partida:
   - 2 jogadores: `1on1`
   - 4 jogadores: `2on2`
   - 6 jogadores: `3on3`
   - 8 jogadores: `4on4`
4. Persistir também os valores brutos de origem em payload, sem sobrescrever a regra normalizada.

Observação:

- `lastscores.mode` e `laststats.mode` continuam úteis como dado bruto de auditoria, mas não devem definir o valor final da coluna `mode`.

### 4. Recorte de campos vindos dos payloads

Campos interessantes confirmados em `laststats`:

- `stats.frags`, `stats.deaths`, `stats.kills`, `stats.tk`, `stats.suicides`, `stats.spawn-frags`
- `dmg.taken`, `dmg.given`, `dmg.self`, `dmg.team`, `dmg.taken-to-die`
- `spree.max`, `spree.quad`
- `weapons.rl.acc.hits`, `weapons.rl.kills.total`
- `weapons.lg.acc.attacks`, `weapons.lg.acc.hits`
- `items.ga.took`, `items.ra.took`, `items.ya.took`, `items.health_100.took`
- `ping`, `team`, `name`, `login`, `xferRL`, `xferLG`

Campos que ficam fora da primeira versão estruturada:

- `control`
- `speed.avg` e `speed.max`
- pickups detalhados por arma
- `health_15`, `health_25`
- damage detalhado por arma além de RL/LG
- cores, skin

Esses continuam acessíveis via `raw_player_payload`.

## Interfaces e Contratos

- O coletor atual não muda de responsabilidade e continua populando somente `collector_matches`.
- O novo job consome `collector_matches.merged_payload`, `score_payload`, `stats_payload`, `has_bots`, `mode`, `map_name`, `played_at`, `server_key`, `demo_name`.
- A camada analítica deve tratar `collector_matches` como fonte de verdade operacional para reprocessamento.
- O dashboard futuro deve consultar `player_match_stats` filtrando `excluded_from_analytics = false`.

## Testes e Cenários

Casos obrigatórios:

- consolida partida duel sem bots em 2 linhas, uma por jogador
- consolida partida 2on2 em 4 linhas, uma por jogador
- marca `excluded_from_analytics = true` quando `has_bots = true`
- mantém `has_bots = true` salvo na fato mesmo quando excluída do dashboard
- aplica regra especial de `dmm4` para `end`, `end2` e `povdmm4`
- usa `2on2`, `3on3`, `4on4` a partir de `lastscores.mode` quando disponível
- calcula `efficiency` corretamente com e sem deaths
- calcula `lg_hit_percentage` corretamente com e sem attacks
- faz `UPSERT` idempotente sem duplicar `(collector_match_id, player_alias_id)`
- cria `players` e `player_aliases` automaticamente para nomes novos
- ao reencontrar alias existente, atualiza `last_seen_at` sem criar duplicata
- reprocessa partida já consolidada quando `collector_matches.updated_at` mudar

Validação integrada:

- rodar stack local existente
- executar coletor
- executar novo job
- verificar consultas agregadas simples por jogador, mapa e modo

## Assumptions

- O nome observado no payload é suficiente para criar a identidade inicial de jogador.
- A vinculação de múltiplos aliases a um player canônico será feita depois via painel administrativo.
- O dashboard futuro deve excluir jogos com bots por filtro de negócio, não por ausência física da linha.
- O arquivo de ideia deve ser criado como novo documento em `docs/ideas`, recomendado: `docs/ideas/consolidacao-player-stats.md`.
- A primeira versão prioriza uma tabela fato “wide” para acelerar analytics e reduzir joins complexos no dashboard.
