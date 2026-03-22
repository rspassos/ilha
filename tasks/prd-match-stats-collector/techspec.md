# Template de Especificação Técnica

## Resumo Executivo

O `match-stats-collector` sera implementado como um binario Go executado por scheduler externo. A solucao fara uma execucao completa por ciclo: carregar a configuracao em YAML, consultar os dois endpoints de cada servidor, correlacionar os registros pelo campo `demo`, enriquecer metadados derivados e persistir o resultado consolidado em PostgreSQL de forma idempotente.

As decisoes centrais sao: usar `pgx/v5` como driver nativo de PostgreSQL, armazenar o payload bruto de `lastscores`, o payload bruto de `laststats` e um payload merged em colunas `JSONB`, e expor observabilidade por logs estruturados e metricas Prometheus. O schema relacional mantera colunas indexaveis para filtros frequentes, incluindo `server_key`, `mode`, `played_at`, `demo_name` e `has_bots`. Para desenvolvimento local, a feature deve incluir um ambiente com Docker Compose contendo um container do coletor e um container de PostgreSQL, usando arquivo `.env` para variaveis sensiveis e de conexao.

## Arquitetura do Sistema

### Visão Geral dos Componentes

- `jobs/collector/cmd/collector`: entrypoint do binario; inicializa config, logger, metrics e ciclo de execucao.
- `jobs/collector/internal/config`: carrega e valida o arquivo YAML de servidores e parametros globais.
- `jobs/collector/internal/httpclient`: encapsula chamadas HTTP, timeouts e decode dos endpoints externos.
- `jobs/collector/internal/collector`: orquestra coleta por servidor, correlacao entre endpoints e tratamento de erros.
- `jobs/collector/internal/merge`: transforma os payloads de score e stats em um `MatchRecord` consolidado.
- `jobs/collector/internal/storage`: persiste registros em PostgreSQL com `UPSERT`.
- `jobs/collector/internal/metrics`: registra contadores, histogramas e gauges Prometheus.
- `docker-compose.yml` ou `compose.yml`: define ambiente local de desenvolvimento com servicos `collector` e `postgres`.
- `.env`: fornece variaveis de ambiente sensiveis e configuracoes locais sem fixa-las no repositorio.

Relacionamentos principais:

- `cmd/collector` depende de `config`, `collector`, `storage`, `metrics`.
- `collector` usa `httpclient` para buscar dados e `merge` para consolidar.
- `storage` recebe `MatchRecord` pronto e executa persistencia idempotente.

Fluxo de dados:

1. O binario inicia e carrega `collector.yaml`.
2. Para cada servidor habilitado, busca `lastscores` e `laststats`.
3. Os registros sao indexados por `demo`.
4. O merge gera um registro unico por partida com campos estruturados e payloads brutos.
5. O storage executa `UPSERT` em PostgreSQL.
6. A execucao publica logs e metricas por servidor e por ciclo.
7. Em desenvolvimento local, os servicos sobem por Docker Compose consumindo variaveis de ambiente do arquivo `.env`.

## Design de Implementação

### Interfaces Principais

```go
type ScoresClient interface {
    FetchLastScores(ctx context.Context, server ServerConfig) ([]ScoreMatch, error)
}

type StatsClient interface {
    FetchLastStats(ctx context.Context, server ServerConfig) ([]StatsMatch, error)
}

type MatchRepository interface {
    UpsertMatches(ctx context.Context, matches []MatchRecord) (UpsertResult, error)
}

type MatchMerger interface {
    Merge(server ServerConfig, scores []ScoreMatch, stats []StatsMatch) ([]MatchRecord, []MergeWarning)
}
```

```go
type CollectorService interface {
    RunOnce(ctx context.Context) error
    CollectServer(ctx context.Context, server ServerConfig) (ServerRunResult, error)
}
```

### Modelos de Dados

Entidades principais:

- `ServerConfig`
  - `key`
  - `name`
  - `base_url`
  - `scores_path`
  - `stats_path`
  - `enabled`
  - `timeout_seconds`

- `MatchRecord`
  - `server_key`
  - `server_name`
  - `demo_name`
  - `match_key`
  - `mode`
  - `map_name`
  - `participants`
  - `played_at`
  - `duration_seconds`
  - `hostname`
  - `has_bots`
  - `score_payload`
  - `stats_payload`
  - `merged_payload`
  - `ingested_at`
  - `updated_at`

- `MergeWarning`
  - `server_key`
  - `demo_name`
  - `reason`

Schema de banco sugerido:

- tabela `collector_matches`
  - `id bigserial primary key`
  - `server_key text not null`
  - `server_name text not null`
  - `demo_name text not null`
  - `match_key text not null`
  - `mode text not null`
  - `map_name text not null`
  - `participants text null`
  - `played_at timestamptz not null`
  - `duration_seconds integer null`
  - `hostname text null`
  - `has_bots boolean not null default false`
  - `score_payload jsonb not null`
  - `stats_payload jsonb null`
  - `merged_payload jsonb not null`
  - `ingested_at timestamptz not null`
  - `updated_at timestamptz not null`

- constraints e indices
  - `unique (server_key, demo_name)`
  - index em `(mode, played_at desc)`
  - index em `(server_key, played_at desc)`
  - index em `(has_bots, played_at desc)`
  - opcional: GIN em `merged_payload`

Regras de modelagem:

- `demo_name` sera o identificador externo principal da partida.
- `match_key` pode repetir `server_key + ":" + demo_name` para simplificar logs e diagnostico.
- `has_bots` sera derivado de qualquer jogador com `is_bot=true` em `lastscores`; se o payload nao trouxer essa chave, o valor padrao sera `false`.
- `stats_payload` pode ser nulo quando apenas um endpoint responder, mas a linha so deve ser persistida se houver identificacao segura por `demo`.

Configuracao por ambiente:

- variaveis sensiveis e de conexao devem ser lidas de variaveis de ambiente
- o projeto deve suportar carregamento local via `.env`
- o binario nao deve embutir credenciais em tempo de compilacao

### Endpoints de API

Nao se aplica. O escopo desta funcionalidade nao inclui API propria.

## Pontos de Integração

- Endpoints HTTP externos por servidor:
  - endpoint de scores dos ultimos 10 jogos
  - endpoint de stats dos ultimos 10 jogos
- Banco PostgreSQL acessivel ao binario por `DATABASE_URL`
- Endpoint local de metricas Prometheus, se habilitado no processo
- Docker Compose para ambiente local de desenvolvimento, com rede compartilhada entre `collector` e `postgres`

Autenticacao:

- nenhuma autenticacao foi identificada nos exemplos atuais; se surgir, deve ser adicionada no modulo `httpclient` sem alterar `collector`.

Tratamento de erros:

- falhas de um servidor nao devem abortar a coleta dos demais
- timeout de request deve ser configuravel por servidor
- respostas invalidas devem ser registradas com contexto do servidor e endpoint
- divergencias entre `lastscores` e `laststats` devem gerar warning, nao panic
- valores sensiveis devem ser fornecidos por ambiente, nao por arquivos versionados

## Abordagem de Testes

### Testes Unidade

Componentes principais a testar:

- parser de YAML e validacao de configuracao
- cliente HTTP com decode de payload
- merge por `demo`
- derivacao de `has_bots`
- derivacao de `played_at`, `mode`, `map_name` e `participants`
- repositorio com montagem de `UPSERT`

Requisitos de mock:

- mock do transporte HTTP
- mock do repositorio apenas nos testes de servico

Cenarios criticos:

- partida presente em ambos endpoints
- partida presente apenas em `lastscores`
- partida presente apenas em `laststats`
- payloads de `duel` e `2on2` com estruturas diferentes
- `has_bots=true` quando qualquer player indicar bot
- duplicidade na mesma execucao e em execucoes consecutivas

### Testes de Integração

- `collector + httpclient + storage` contra PostgreSQL local ou containerizado
- fixture baseada em [lastscores.json](/home/rspassos/projects/ilha/docs/responses/lastscores.json) e [laststats.json](/home/rspassos/projects/ilha/docs/responses/laststats.json)
- verificar `UPSERT`, constraint unica, persistencia de JSONB e filtro por `has_bots`

### Testes de E2E

Nao se aplica. Nao ha frontend nem fluxo Playwright neste escopo.

## Sequenciamento de Desenvolvimento

### Ordem de Construção

1. Criar estrutura do job em `jobs/collector/` com entrypoint, config e logging.
2. Definir modelos de payload e parser dos endpoints com fixtures reais.
3. Implementar merge por `demo` e regras derivadas como `has_bots`.
4. Implementar schema e repositorio PostgreSQL com `UPSERT`.
5. Integrar ciclo `RunOnce` por servidor e tratamento parcial de falhas.
6. Adicionar metricas Prometheus e testes de integracao.
7. Adicionar ambiente local com Docker Compose, `.env.example` e documentacao operacional.

### Dependências Técnicas

- disponibilidade de instância PostgreSQL
- definicao do arquivo YAML de servidores
- acessibilidade de rede aos endpoints de origem a partir do ambiente de execucao
- Docker e Docker Compose disponiveis para desenvolvimento local

## Monitoramento e Observabilidade

Metricas Prometheus:

- `collector_runs_total{status}`
- `collector_server_runs_total{server_key,status}`
- `collector_matches_fetched_total{server_key,source}`
- `collector_matches_upserted_total{server_key,result}`
- `collector_merge_warnings_total{server_key,reason}`
- `collector_request_duration_seconds{server_key,endpoint}`

Logs estruturados:

- inicio e fim de execucao
- inicio e fim por servidor
- status HTTP, duracao e quantidade de itens por endpoint
- quantidade de partidas merged, inserted e updated
- warnings de correlacao e erros de persistencia

Niveis de log:

- `INFO` para execucao normal
- `WARN` para mismatch, payload parcial e servidor indisponivel
- `ERROR` para falha de request, decode ou banco

Nao ha evidencia de dashboards Grafana existentes no repositorio atual.

## Considerações Técnicas

### Decisões Principais

- PostgreSQL com `JSONB` foi escolhido porque combina flexibilidade de payload com filtros e indices relacionais. A documentacao oficial do PostgreSQL destaca operadores e indexacao especificos para `jsonb`, o que sustenta essa abordagem.
- `pgx/v5` e preferivel ao `database/sql` genérico porque o projeto tem alvo exclusivo em PostgreSQL e `pgx` oferece suporte nativo a `jsonb`, pooling e melhor controle operacional.
- O parser YAML deve usar `go.yaml.in/yaml/v4`, que e o fork mantido pela organizacao YAML; `go-yaml/yaml` foi arquivado em abril de 2025.
- As metricas devem usar `prometheus/client_golang`, biblioteca oficial de instrumentacao em Go.
- Scheduler externo foi escolhido para manter o binario stateless e simplificar operacao, retries e agendamento.
- O ambiente local deve usar Docker Compose com um container do coletor e um container PostgreSQL para reduzir friccao de onboarding e padronizar desenvolvimento.
- Credenciais e URLs devem ser injetadas por variaveis de ambiente; embutir segredos no binario foi rejeitado por fragilidade operacional e risco de vazamento.
- A implementacao deve seguir a skill [golang-patterns](/home/rspassos/projects/ilha/.agents/skills/golang-patterns/SKILL.md), com enfase em simplicidade, uso consistente de `context.Context`, erros com contexto, interfaces pequenas e design idiomatico de pacotes Go.

Trade-offs:

- Persistir payload bruto aumenta armazenamento, mas reduz risco de perder sinal util para consumidores futuros.
- Persistir uma linha mesmo sem `stats_payload` completo aumenta tolerancia a falhas, mas exige consumers atentos a campos nulos.
- Nao criar tabela de jogadores no MVP simplifica a entrega, mas desloca certas agregacoes para uma fase posterior.
- Rodar o coletor em container local facilita reproducao, mas adiciona arquivos de infraestrutura e dependencia de Docker no fluxo de desenvolvimento.

Alternativas rejeitadas:

- MongoDB como armazenamento primario: rejeitado no MVP por pior alinhamento com filtros estruturados e futuras consultas relacionais.
- Processo residente com ticker interno: rejeitado em favor de binario acionado por scheduler externo.
- Compilar segredos junto com o binario: rejeitado porque dificulta rotacao, separacao por ambiente e distribuicao segura.

### Riscos Conhecidos

- A API expor apenas os ultimos 10 jogos cria risco real de perda de dados se a frequencia do scheduler for baixa.
- A correlacao por `demo` assume estabilidade desse campo entre endpoints; se isso mudar, a estrategia de merge quebra.
- Variacao de payload entre modos pode introduzir casos nao cobertos nos parsers.
- `is_bot` aparece nos exemplos de `lastscores`; se faltar em algum modo ou servidor, `has_bots` pode subnotificar.
- Configuracao inconsistente entre `.env`, Compose e YAML pode causar falhas locais dificeis de diagnosticar.

Mitigacoes:

- definir janela de execucao inicial de 1 minuto e revisar com dados reais
- preservar payload bruto para permitir correcoes retroativas de parser
- adicionar fixtures reais por modo de jogo
- registrar warnings de match parcial por `demo`
- fornecer `.env.example`, validacao de configuracao no startup e erros claros para variaveis obrigatorias ausentes

### Conformidade com Skills Padrões

- Skill obrigatoria: [golang-patterns](/home/rspassos/projects/ilha/.agents/skills/golang-patterns/SKILL.md)
- Aplicacao nesta tech spec:
  - manter pacotes pequenos e com responsabilidade clara
  - aceitar interfaces e retornar tipos concretos quando fizer sentido
  - propagar `context.Context` em chamadas IO-bound
  - envolver erros com contexto suficiente para operacao e debug
  - evitar complexidade desnecessaria e abstrações precoces

### Arquivos relevantes e dependentes

- [tasks/prd-match-stats-collector/prd.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/prd.md)
- [docs/responses/lastscores.json](/home/rspassos/projects/ilha/docs/responses/lastscores.json)
- [docs/responses/laststats.json](/home/rspassos/projects/ilha/docs/responses/laststats.json)
- [templates/techspec-template.md](/home/rspassos/projects/ilha/templates/techspec-template.md)
- [tasks/prd-match-stats-collector/techspec.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/techspec.md)
- [.agents/skills/golang-patterns/SKILL.md](/home/rspassos/projects/ilha/.agents/skills/golang-patterns/SKILL.md)
