# Ideia: Coleta de Stats de Partidas em Multiplos Servidores

## Contexto

Existe uma API com dois endpoints por servidor:

- um endpoint que retorna os resultados dos ultimos 10 jogos
- um endpoint que retorna os stats dos ultimos 10 jogos

O objetivo e coletar dados de varios servidores continuamente e armazenar tudo de forma historica para consulta futura.

## Ponto mais importante

O maior risco dessa arquitetura nao e o banco. E o fato de a API expor apenas os ultimos 10 jogos.

Isso significa que, se um servidor tiver alta rotacao e o job rodar com pouca frequencia, voce vai perder partidas entre uma execucao e outra.

Entao a solucao precisa garantir:

- polling frequente
- deduplicacao
- armazenamento idempotente
- preservacao do payload bruto

## Recomendacao objetiva

Eu nao comecaria com um banco puramente nao relacional.

Minha recomendacao inicial seria:

- job em Go
- execucao via cronjob ou scheduler simples
- PostgreSQL com colunas estruturadas + `JSONB` para o payload variavel

Motivo:

- voce provavelmente vai querer filtrar por servidor, tipo de partida, mapa, data, jogador e resultado
- PostgreSQL resolve bem consultas historicas e agregacoes
- `JSONB` permite guardar diferencas entre `duel`, `2on2` e `4on4` sem travar a modelagem
- e mais simples operar Postgres do que introduzir MongoDB sem necessidade clara

Se no futuro o payload variar muito e a consulta for quase sempre por documento inteiro, ai MongoDB pode fazer sentido. Para MVP, eu iria de Postgres.

## Arquitetura sugerida

### 1. Configuracao de servidores

Manter uma lista de servidores em um arquivo ou tabela:

- `server_id`
- `name`
- `base_url`
- `enabled`
- `poll_interval_seconds`

No inicio, um arquivo YAML ou JSON ja basta.

### 2. Job coletor em Go

Um processo que:

1. percorre a lista de servidores
2. chama os 2 endpoints de cada servidor
3. correlaciona resultado + stats da mesma partida
4. gera um identificador unico da partida
5. faz upsert no banco
6. registra logs e metricas

Esse job pode rodar de duas formas:

- cron a cada 1 minuto
- processo continuo com ticker por servidor

Se os servidores tiverem ritmos diferentes, eu prefiro processo continuo com ticker. Se quiser simplicidade operacional, cronjob por minuto funciona bem no comeco.

## Fluxo de coleta

Para cada servidor:

1. Buscar `last_results`
2. Buscar `last_stats`
3. Fazer merge dos itens
4. Para cada partida:
   - identificar `match_id`
   - normalizar campos comuns
   - guardar payload bruto
   - fazer upsert

## Como identificar a mesma partida

Ideal:

- a API ja retorna um `match_id` unico

Se nao retornar, monte uma chave derivada, por exemplo:

- `server_id`
- `finished_at` ou `started_at`
- `map`
- `game_type`
- hash dos players/teams

Exemplo de chave:

`server_id + finished_at + game_type + map + players_hash`

Sem uma chave consistente, voce vai sofrer com duplicidade ou sobrescrita incorreta.

## Modelagem recomendada

### Tabela `servers`

Campos:

- `id`
- `name`
- `base_url`
- `enabled`
- `poll_interval_seconds`
- `created_at`
- `updated_at`

### Tabela `matches`

Campos comuns e indexaveis:

- `id`
- `server_id`
- `external_match_id` nullable
- `match_key` unique
- `game_type` (`duel`, `2on2`, `4on4`)
- `map_name`
- `started_at` nullable
- `finished_at`
- `winner`
- `status`
- `raw_result_json JSONB`
- `raw_stats_json JSONB`
- `raw_merged_json JSONB`
- `ingested_at`
- `updated_at`

### Tabela opcional `match_players`

Se voce quiser analytics de jogadores com mais facilidade, vale extrair uma estrutura minima:

- `match_id`
- `player_name`
- `player_id` nullable
- `team`
- `score`
- `frags`
- `deaths`
- `damage_given`
- `damage_taken`
- `raw_player_json JSONB`

Essa abordagem hibrida costuma ser a melhor:

- o documento completo fica salvo
- os campos mais consultados ficam estruturados

## Por que nao so MongoDB?

MongoDB faz sentido se:

- o schema muda muito
- voce quase nao faz joins
- voce quer iterar muito rapido em cima do payload cru

Mas eu ainda vejo algumas desvantagens para esse caso:

- relatorios historicos e agregacoes podem ficar mais chatos
- controle transacional e upsert com regras de unicidade costuma ficar mais direto em Postgres
- se depois voce quiser dashboard SQL, BI ou consultas ad hoc, Postgres ajuda mais

## Estrategia de polling

Como a API so retorna os ultimos 10 jogos, a frequencia precisa ser definida com base no servidor mais movimentado.

Exemplo pratico:

- se um servidor consegue fechar mais de 10 partidas em 5 minutos, nao adianta rodar a cada 5 minutos
- nesse caso, rode a cada 30 segundos ou 1 minuto

Regra pragmatica para MVP:

- comecar com coleta a cada 1 minuto
- medir quantas partidas novas entram por ciclo
- se aparecer gap, reduzir para 30 segundos

## Deduplicacao e idempotencia

O job precisa ser seguro para reprocessar os mesmos ultimos 10 jogos varias vezes.

Entao use:

- `UPSERT`
- constraint unica em `match_key`
- atualizacao apenas dos campos que podem chegar depois

Isso e importante porque, dependendo da API, o endpoint de resultado e o de stats podem nao refletir exatamente o mesmo estado no mesmo instante.

## Observabilidade

Desde o inicio, eu colocaria:

- log por servidor
- quantidade de partidas lidas por execucao
- quantidade de inserts
- quantidade de updates
- quantidade de erros por endpoint
- tempo de resposta por servidor

Se quiser subir um pouco o nivel:

- endpoint `/metrics` com Prometheus

## Proposta de MVP

### Fase 1

- job em Go
- lista de servidores em arquivo
- polling a cada 1 minuto
- Postgres
- tabela `matches` com campos comuns + `JSONB`
- deduplicacao por `match_key`

### Fase 2

- extrair `match_players`
- criar dashboard
- metricas e alertas
- retry/backoff por servidor

### Fase 3

- processamento assincorno com fila
- particionamento por data no banco
- agregacoes materializadas

## Alternativa ainda mais simples

Se quiser validar rapido antes de modelar bem:

- salvar somente o payload merged em uma tabela `matches_raw`
- extrair apenas:
  - `server_id`
  - `match_key`
  - `game_type`
  - `finished_at`
  - `payload JSONB`

Depois, com dados reais em maos, voce decide o que vale normalizar.

Essa abordagem reduz risco de modelar cedo demais.

## Minha recomendacao final

Se eu fosse implementar hoje, faria assim:

1. Go worker
2. scheduler simples a cada 1 minuto
3. PostgreSQL
4. tabela principal com colunas comuns + `JSONB`
5. deduplicacao forte por `match_key`
6. payload bruto sempre preservado

Isso te da:

- simplicidade operacional
- flexibilidade para payloads diferentes
- boa capacidade de consulta
- caminho claro para evoluir sem jogar fora o MVP

## Perguntas que valem validar antes de implementar

- os endpoints retornam algum `match_id` estavel?
- os dois endpoints usam exatamente a mesma ordenacao?
- existe risco de um endpoint trazer uma partida e o outro ainda nao?
- qual o maior volume de partidas por servidor por minuto?
- voce quer analytics por jogador desde o inicio ou so armazenar historico bruto?

## Estrutura sugerida no codigo

```text
cmd/collector/main.go
internal/config/
internal/client/
internal/collector/
internal/storage/
internal/model/
```

Responsabilidades:

- `client`: chama a API dos servidores
- `collector`: orquestra merge, dedupe e ingestao
- `storage`: persiste no banco
- `model`: structs comuns e payloads

## Decisao curta

Se a duvida principal for "banco relacional ou nao relacional?", minha resposta e:

- use Postgres com `JSONB` no MVP
- nao va direto para MongoDB so porque o JSON muda por tipo de partida

Variacao de schema, sozinha, nao e motivo suficiente para abandonar relacional quando voce ainda quer historico, filtros e agregacoes.
