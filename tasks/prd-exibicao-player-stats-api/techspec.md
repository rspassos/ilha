# Template de Especificação Técnica

## Resumo Executivo

A solucao sera um servico HTTP Go separado no repositorio, dedicado a expor um contrato publico de ranking de jogadores a partir da camada analitica ja existente em `player_match_stats`, `player_canonical` e `player_aliases`. A primeira versao atendera apenas ranking publico, sem perfis, historico de partidas ou dashboards, com ordenacao padrao por `efficiency`, filtros por modo, mapa, servidor e periodo, exclusao permanente de partidas com `excluded_from_analytics = true`, minimo de 10 partidas por jogador e paginacao por `limit` e `offset`.

As decisoes principais sao: manter os jobs batch inalterados; nao criar nova camada de pre-agregacao neste ciclo; implementar um endpoint unico de leitura com ordenacao parametrizada e whitelist de campos; reutilizar os padroes atuais do repositorio para config, logger JSON, Prometheus e acesso PostgreSQL; e adotar respostas de erro em `application/problem+json`, alinhadas ao RFC 9457. A paginacao pode expor metadados no corpo e, opcionalmente, `Link` headers compativeis com RFC 8288.

## Arquitetura do Sistema

### Visão Geral dos Componentes

- `services/player-stats-api/cmd/player-stats-api`: entrypoint do novo binario HTTP.
- `services/player-stats-api/internal/config`: carrega env vars do servico, incluindo bind HTTP, bind de metricas, `DATABASE_URL`, defaults de paginacao e minimo de partidas.
- `services/player-stats-api/internal/bootstrap`: inicializa logger, pool PostgreSQL, repositorio, servico de ranking e servidores HTTP.
- `services/player-stats-api/internal/httpapi`: handlers, roteamento com `net/http`, serializacao JSON, validacao de query params e respostas de erro.
- `services/player-stats-api/internal/service`: aplica regras de dominio de leitura, whitelist de ordenacao, defaults e montagem da resposta.
- `services/player-stats-api/internal/storage`: executa consultas SQL agregadas sobre `player_match_stats` e join com `player_canonical`.
- `services/player-stats-api/internal/metrics`: expoe metricas Prometheus do endpoint HTTP.

Relacionamentos principais:

- `cmd` depende de `bootstrap`.
- `bootstrap` conecta `config`, `logging`, `metrics`, `storage`, `service` e `httpapi`.
- `httpapi` chama `service` para resolver a consulta de ranking.
- `service` chama `storage` e nao acessa SQL diretamente.
- `storage` trata `player_match_stats` como fato base e `player_canonical` como dimensao de exibicao.

Fluxo de dados:

1. O cliente chama o endpoint de ranking com filtros e ordenacao opcionais.
2. O handler valida os parametros, aplica defaults e rejeita campos de ordenacao fora da whitelist.
3. O servico monta a consulta de dominio com `minimum_matches = 10` e `excluded_from_analytics = false`.
4. O repositorio executa SQL agregada, pagina por `limit` e `offset`, e retorna linhas ranqueadas.
5. O handler responde JSON com `data` e `meta` da consulta; erros de validacao ou falha interna usam `application/problem+json`.

## Design de Implementação

### Interfaces Principais

```go
type RankingService interface {
    ListPlayerRanking(ctx context.Context, query RankingQuery) (RankingPage, error)
}
```

```go
type RankingRepository interface {
    ListPlayerRanking(ctx context.Context, query RankingQuery) (RankingPage, error)
}
```

```go
type ProblemWriter interface {
    WriteProblem(w http.ResponseWriter, status int, problem Problem)
}
```

### Modelos de Dados

Entidades e tipos principais:

- `RankingQuery`
  - `Mode string`
  - `Map string`
  - `Server string`
  - `From time.Time`
  - `To time.Time`
  - `SortBy string`
  - `SortDirection string`
  - `Limit int`
  - `Offset int`
  - `MinimumMatches int`

- `PlayerRankingRow`
  - `player_id`
  - `display_name`
  - `matches`
  - `efficiency`
  - `frags`
  - `kills`
  - `deaths`
  - `lg_accuracy`
  - `rl_hits`
  - `rank`

- `RankingPage`
  - `Data []PlayerRankingRow`
  - `Meta PaginationMeta`
  - `Filters AppliedFilters`

- `PaginationMeta`
  - `limit`
  - `offset`
  - `returned`
  - `has_next`

Regras de agregacao:

- agregar sempre sobre `player_match_stats`
- join com `player_canonical` por `player_id`
- filtrar sempre `excluded_from_analytics = false`
- aplicar `HAVING count(*) >= 10`
- usar whitelist de ordenacao: `efficiency`, `frags`, `lg_accuracy`, `rl_hits`
- default de ordenacao: `efficiency desc`
- desempate fixo: `frags desc`, `matches desc`, `player_id asc`

Esquema de banco:

- nao ha mudanca obrigatoria de tabela para v1
- indice novo recomendado:
  - `player_match_stats_rank_filters_idx` em `(excluded_from_analytics, normalized_mode, server_key, map_name, played_at desc)`
- se o volume crescer, revisar com `EXPLAIN ANALYZE` antes de adicionar indices mais especificos

### Endpoints de API

- `GET /v1/rankings/players`
  - Retorna ranking publico de jogadores
  - Query params suportados:
    - `mode`
    - `map`
    - `server`
    - `from`
    - `to`
    - `sort_by`
    - `sort_direction`
    - `limit`
    - `offset`
  - Resposta 200:
    - `data`: lista de jogadores ranqueados
    - `meta`: paginacao e defaults aplicados
    - `filters`: recorte efetivo da consulta
  - Resposta 400:
    - problem details para filtro invalido, data invalida, limite fora da faixa ou `sort_by` nao permitido
  - Resposta 500:
    - problem details para falha interna

Formato de resposta esperado:

- `data` deve trazer apenas campos publicos e agregados
- `meta` deve incluir `sort_by`, `sort_direction`, `minimum_matches`, `limit`, `offset` e `has_next`
- `filters` deve refletir os valores normalizados aceitos pela API

## Pontos de Integração

- PostgreSQL local do projeto
  - leitura de `player_match_stats` e `player_canonical`
  - sem dependencia de servicos externos adicionais
- Site existente
  - integra via HTTP JSON publico
  - sem autenticacao na v1

Tratamento de erros:

- validacao de entrada retorna `application/problem+json`, conforme RFC 9457
- pagina seguinte e anterior podem ser comunicadas no corpo e opcionalmente com `Link` headers segundo RFC 8288
- timeouts de leitura devem ser configurados no servidor HTTP e no contexto de banco

## Abordagem de Testes

### Testes Unidade

- validacao de query params no handler
- normalizacao de defaults (`sort_by`, `sort_direction`, `limit`, `offset`)
- rejeicao de `sort_by` fora da whitelist
- serializacao de `problem+json`
- montagem da resposta quando nao houver resultados

Mocks:

- mock apenas do servico no teste de handler
- mock apenas do repositorio no teste do servico

### Testes de Integração

- `storage + PostgreSQL` consultando fixtures geradas pelo fluxo real do repositorio
- ranking sem filtros com ordenacao por `efficiency desc`
- ranking com filtros combinados por `mode`, `map`, `server` e periodo
- exclusao de linhas com `excluded_from_analytics = true`
- aplicacao de `HAVING count(*) >= 10`
- ordenacao secundaria deterministica em empates
- paginacao por `limit` e `offset`

### Testes de E2E

Nao se aplica neste ciclo, porque o escopo aprovado cobre apenas a API e nao inclui alteracao de frontend.

## Sequenciamento de Desenvolvimento

### Ordem de Construção

1. Criar o esqueleto do novo servico `services/player-stats-api` com `cmd`, `config`, `bootstrap`, `logging` e `metrics`, reaproveitando os padroes dos jobs existentes.
2. Definir contratos HTTP, tipos de consulta e estrategia de validacao, porque isso trava handler, servico e repositorio.
3. Implementar repositorio SQL de ranking e validar a consulta principal com teste de integracao.
4. Implementar servico de dominio com defaults, whitelist de ordenacao e regras de filtro.
5. Implementar handlers HTTP, erros `problem+json` e paginacao.
6. Integrar ao `compose.yml`, README local e metricas.
7. Fechar testes de unidade e integracao.

### Dependências Técnicas

- PostgreSQL acessivel com schema de `player-stats` aplicado
- `player_match_stats` populada pelo job `player-stats`
- novo servico incluido no `compose.yml` para validacao local
- ausencia de `.claude/rules` no repositorio atual; nenhuma regra adicional foi encontrada para este ciclo

## Monitoramento e Observabilidade

Metricas Prometheus:

- `player_stats_api_requests_total{endpoint,status}`
- `player_stats_api_request_duration_seconds{endpoint}`
- `player_stats_api_ranking_rows_returned_total`
- `player_stats_api_invalid_requests_total{reason}`
- `player_stats_api_db_queries_total{query,status}`

Os nomes seguem as recomendacoes de prefixo e baixo cardinalidade do Prometheus. Nao usar labels com `player_id`, `map` ou `server`, para evitar cardinalidade alta.

Logs estruturados:

- inicio do servico com bind HTTP, bind de metricas e defaults
- request concluida com endpoint, status, duracao, filtros relevantes e tamanho da resposta
- erros de validacao com motivo sintetico
- erros de banco com operacao e duracao

Nao ha evidencia de dashboards Grafana existentes no repositorio; a entrega deve expor apenas endpoint Prometheus compativel.

## Considerações Técnicas

### Decisões Principais

- Servico HTTP separado foi escolhido para nao acoplar trafego publico aos jobs batch existentes.
- `net/http` deve ser preferido na v1, porque o repositorio ainda nao usa framework HTTP e o escopo de endpoints e pequeno.
- Leitura direta da tabela fato foi escolhida em vez de nova tabela agregada, porque o primeiro caso de uso e um ranking simples e a escala esperada e baixa a media.
- Ordenacao dinamica sera controlada por whitelist de aliases SQL, evitando interpolacao livre de coluna e risco de injecao.
- `limit` e `offset` foram escolhidos por simplicidade de integracao com o site; keyset pagination fica como evolucao futura se o volume crescer.

Trade-offs e alternativas rejeitadas:

- reutilizar `jobs/player-stats` como servidor HTTP foi rejeitado porque mistura batch e servico always-on no mesmo binario.
- adicionar framework como `chi` ou `gin` foi rejeitado porque nao ha uso previo no repositorio e o ganho inicial e baixo.
- criar materialized view ou tabela de ranking foi rejeitado neste ciclo para evitar complexidade prematura e duplicacao de logica.

### Riscos Conhecidos

- jogadores com exatamente 10 partidas podem entrar e sair do ranking com frequencia conforme novos filtros; mitigacao: expor `minimum_matches` em `meta`
- `limit` e `offset` degradam em paginas muito profundas; mitigacao: aceitar isso na v1 e revisar se o uso real justificar cursor
- ordenacao por metricas agregadas pode gerar empates frequentes; mitigacao: desempate deterministico fixo
- filtros textuais por mapa e servidor podem exigir normalizacao de casing; mitigacao: definir comparacao consistente e documentada na API
- o indice atual pode nao cobrir todos os filtros compostos; mitigacao: validar a query principal com `EXPLAIN ANALYZE` e adicionar um indice composto apenas se necessario

### Conformidade com Skills Padrões

- `golang-patterns`: aplicavel para manter interfaces pequenas, composicao simples e uso idiomatico de `context`, `net/http` e `pgx`
- `supabase-postgres-best-practices`: aplicavel para revisar indice composto, agregacao e custo da consulta principal no PostgreSQL

### Arquivos relevantes e dependentes

- `/home/rspassos/projects/ilha/tasks/prd-exibicao-player-stats-api/prd.md`
- `/home/rspassos/projects/ilha/templates/techspec-template.md`
- `/home/rspassos/projects/ilha/jobs/player-stats/internal/storage/migrations/001_create_player_stats_analytics.sql`
- `/home/rspassos/projects/ilha/jobs/player-stats/internal/model/analytics.go`
- `/home/rspassos/projects/ilha/jobs/player-stats/internal/bootstrap/bootstrap.go`
- `/home/rspassos/projects/ilha/jobs/player-stats/internal/metrics/metrics.go`
- `/home/rspassos/projects/ilha/jobs/player-stats/internal/logging/logger.go`
- `/home/rspassos/projects/ilha/compose.yml`
