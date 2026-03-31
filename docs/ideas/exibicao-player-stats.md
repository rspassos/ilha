# Plano: Exibicao de Player Stats

## Resumo

A recomendacao para v1 e nao criar outra rotina batch nem outra tabela consolidada agora. O projeto ja tem a camada certa para consumo analitico em `player_match_stats`, com `player_canonical` e `player_aliases` para identidade.

Para o primeiro caso de uso, a melhor opcao e criar uma API de leitura que consulte diretamente essas tabelas e entregue payloads proprios para o frontend. Isso mantem o pipeline simples, evita duplicar logica de negocio e deixa a atualizacao praticamente no mesmo ritmo do job `player-stats`.

A segunda camada de pre-agregacao so passa a valer a pena quando houver pelo menos um destes sinais:

- rankings globais pesados com filtros arbitrarios comecarem a ficar lentos
- dashboard inicial exigir muitas agregacoes repetidas na mesma tela
- volume crescer a ponto de scans e aggregations frequentes em `player_match_stats` ficarem caros

## Mudancas de Implementacao

### 1. Criar um servico API de leitura separado

Implementar um servico HTTP pequeno, isolado dos jobs batch, com responsabilidade apenas de consulta.

Contratos iniciais recomendados:

- `GET /players?mode=&map=&server=&from=&to=&sort=&page=`
  Retorna ranking ou listagem de jogadores com metricas agregadas no periodo.
- `GET /players/:player_id`
  Retorna dados canonicos do jogador e aliases observados.
- `GET /players/:player_id/matches?mode=&map=&server=&from=&to=&page=`
  Retorna historico de partidas do jogador.
- `GET /players/:player_id/summary?mode=&map=&server=&from=&to=`
  Retorna agregados do jogador no recorte filtrado.

A API deve:

- ler de `player_match_stats` como fonte principal
- fazer join com `player_canonical` para nome canonico
- excluir por padrao `excluded_from_analytics = true`
- aceitar filtro explicito para incluir bots depois, se necessario
- devolver DTOs proprios, sem expor a estrutura SQL diretamente ao frontend

### 2. Tratar `player_match_stats` como fato base

Usar a tabela atual como source of truth da exibicao.

Consultas suportadas bem pela modelagem atual:

- historico de partidas por jogador
- ranking por periodo, modo, mapa e servidor
- perfil do jogador com medias, totais e ultimas partidas
- filtros por `normalized_mode`, `map_name`, `server_key` e `played_at`

Ajustes de schema so se a leitura mostrar necessidade real:

- indice composto para filtros mais frequentes, por exemplo `(excluded_from_analytics, normalized_mode, played_at desc)`
- indice por `(player_id, normalized_mode, played_at desc)` se perfil por modo for muito usado

### 3. Reservar uma camada agregada para fase 2

Nao criar outra tabela batch na v1. Planejar uma evolucao explicita para quando houver necessidade.

Opcao futura preferida:

- tabela derivada ou materialized view de agregados por periodo fixo, por exemplo diario ou semanal por jogador e modo

Usar essa camada so para:

- leaderboards globais muito acessados
- cards e KPIs repetidos em dashboard geral
- series temporais agregadas para multiplos jogadores e servidores

Evitar desde ja:

- duplicar todos os campos de `player_match_stats` em outra tabela
- consolidar demais antes de saber quais queries o frontend realmente fara
- colocar logica de apresentacao dentro do job `player-stats`

### 4. Interface e regra de negocio

Defaults da API:

- filtro padrao `excluded_from_analytics = false`
- ordenacao padrao de ranking por uma metrica definida de negocio, por exemplo `frags desc` ou `efficiency desc`
- paginacao obrigatoria em listagens
- recorte temporal explicito para queries agregadas

Metricas derivadas da API devem ser calculadas sobre o fato base:

- totais: `sum(frags)`, `sum(kills)`, `sum(deaths)` e similares
- medias: `avg(efficiency)`, `avg(lg_accuracy)` e similares
- contagens: `count(*)` de partidas validas no filtro

## Testes e Cenarios

Validacao tecnica:

- confirmar que ranking com filtros por `mode`, `map`, `server` e periodo responde com latencia aceitavel usando SQL direto
- validar paginacao estavel por ordenacao deterministica
- confirmar exclusao padrao de `excluded_from_analytics = true`
- validar joins de identidade sem duplicar jogador por alias

Cenarios de produto:

- listar top jogadores de `2on2` em um periodo
- abrir perfil de um jogador e ver resumo agregado
- abrir historico do jogador e navegar por partidas
- filtrar por mapa e servidor mantendo os mesmos totais esperados

Criterio para introduzir pre-agregacao:

- adicionar outra tabela ou materializacao apenas se consultas principais ficarem lentas com indices razoaveis
- ou se a mesma agregacao for recalculada muitas vezes por trafego normal
- ou se o dashboard exigir agregados amplos demais para serem montados on demand com custo aceitavel

## Assumptions

- O primeiro uso da UI e perfil ou ranking, nao um dashboard executivo pesado.
- A escala inicial e baixa a media, entao SQL sobre `player_match_stats` e suficiente.
- Atualizacao quase em tempo real significa refletir o ultimo batch do job `player-stats`, sem nova etapa de consolidacao.
- A melhor arquitetura inicial e:
  1. `collector_matches` como bruto operacional
  2. `player_match_stats` como fato analitico
  3. API de leitura para o frontend
  4. agregados materializados apenas se a observacao de uso justificar
