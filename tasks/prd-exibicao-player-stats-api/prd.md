# PRD: API Publica de Ranking de Jogadores

## Visão Geral

O projeto ja possui uma base analitica de estatisticas por jogador derivada das partidas coletadas. O proximo passo e disponibilizar esses dados por meio de uma API publica que permita ao site existente consumir um ranking de jogadores sem depender de leitura direta do banco.

O problema atual e a ausencia de uma interface publica e estavel para expor o ranking. Embora a base analitica ja exista, o site ainda nao possui um contrato de consumo proprio para listar jogadores, aplicar filtros de contexto e ordenar resultados de forma consistente.

O valor desta funcionalidade esta em criar uma camada de acesso publica, previsivel e focada em ranking, que permita evoluir a experiencia do site sem acoplar o frontend ao schema interno do banco ou aos jobs batch que produzem os dados.

## Objetivos

- Disponibilizar uma API publica para consulta de ranking de jogadores.
- Permitir que o site existente consuma o ranking sem depender de acesso direto ao banco.
- Garantir que o ranking reflita apenas partidas elegiveis para analytics.
- Permitir filtragem por recortes relevantes de jogo para o ranking publico.
- Estabelecer um contrato estavel para evolucao futura da exibicao no site.

Como e o sucesso:

- O site consegue listar jogadores ranqueados consumindo apenas a API.
- O ranking e ordenado por eficiencia como criterio padrao.
- Jogos marcados como excluidos de analytics nao aparecem no ranking.
- O consumidor da API consegue aplicar filtros suportados sem interpretar estruturas internas do banco.

Metricas principais para acompanhar:

- quantidade de chamadas ao endpoint de ranking
- percentual de respostas bem-sucedidas do endpoint
- tempo de resposta do endpoint de ranking
- quantidade media de jogadores retornados por consulta
- percentual de consultas com filtros aplicados

Objetivos de negocio:

- habilitar a primeira exibicao publica de estatisticas de jogadores no site
- reduzir acoplamento entre frontend e estrutura interna de dados
- criar base contratual para futuras evolucoes de exibicao publica

## Histórias de Usuário

- Como visitante do site, eu quero visualizar um ranking de jogadores para identificar quem performa melhor.
- Como visitante do site, eu quero que o ranking use um criterio padrao claro e consistente para comparar jogadores.
- Como visitante do site, eu quero filtrar o ranking por contexto de jogo para consultar recortes mais relevantes.
- Como time responsavel pelo site, eu quero consumir uma API estavel para exibir o ranking sem conhecer detalhes internos do banco.
- Como time de produto, eu quero garantir que partidas excluidas de analytics nao contaminem o ranking publico.

Personas principais:

- Visitante publico do site
- Time responsavel pelo frontend do site

Casos extremos relevantes:

- nao existir volume suficiente de partidas para alguns filtros
- um jogador ter poucos jogos e ainda assim aparecer no ranking
- filtros combinados retornarem lista vazia
- o ranking precisar permanecer consistente mesmo com atualizacoes periodicas da base analitica

## Funcionalidades Principais

### 1. Exposicao publica do ranking de jogadores

A funcionalidade deve disponibilizar uma consulta publica de ranking baseada nos dados analiticos ja consolidados.

Por que e importante:

- viabiliza o consumo pelo site existente
- separa a exibicao publica da estrutura interna de armazenamento
- cria um contrato proprio para evolucao futura

Requisitos funcionais:

1. O sistema deve disponibilizar um endpoint publico para consulta de ranking de jogadores.
2. O sistema deve retornar uma lista de jogadores ordenada para exibicao no site.
3. O sistema deve expor no ranking, no minimo, a identificacao publica do jogador e as metricas necessarias para justificar sua posicao.
4. O sistema deve retornar dados em formato consistente e apropriado para consumo por frontend.

### 2. Ordenacao padrao por eficiencia

O ranking inicial deve usar eficiencia como criterio padrao de ordenacao.

Por que e importante:

- define uma regra clara para a primeira experiencia publica
- evita ambiguidade sobre a ordem exibida ao usuario final

Requisitos funcionais:

5. O sistema deve ordenar o ranking por eficiencia como criterio padrao.
6. O sistema deve manter o mesmo criterio padrao de ordenacao em todas as consultas sem ordenacao explicita.
7. O sistema deve retornar a eficiencia de forma explicita para cada jogador listado.

### 3. Filtros de ranking

O ranking deve permitir recortes que tornem a consulta mais util ao publico e ao site.

Por que e importante:

- aumenta a relevancia da informacao exibida
- permite contextualizar a comparacao entre jogadores

Requisitos funcionais:

8. O sistema deve permitir filtrar o ranking por modo de jogo.
9. O sistema deve permitir filtrar o ranking por mapa.
10. O sistema deve permitir filtrar o ranking por servidor.
11. O sistema deve permitir filtrar o ranking por periodo.
12. O sistema deve aplicar os filtros de forma combinavel dentro da mesma consulta.

### 4. Exclusao de partidas fora do consumo analitico

O ranking publico deve refletir apenas partidas consideradas validas para analytics.

Por que e importante:

- protege a credibilidade do ranking
- evita exposicao publica de dados que o proprio dominio ja classificou como invalidos para analise

Requisitos funcionais:

13. O sistema nao deve incluir no ranking partidas marcadas como excluidas de analytics.
14. O sistema deve ocultar permanentemente esses dados do consumo publico nesta primeira versao.
15. O sistema deve garantir que o ranking publico use apenas a base elegivel para analytics.

### 5. Consumo estavel pelo site existente

Esta funcionalidade deve priorizar previsibilidade para o frontend que ja existe.

Por que e importante:

- reduz retrabalho de integracao
- facilita evolucao incremental do site

Requisitos funcionais:

16. O sistema deve fornecer um contrato publico estavel para o ranking de jogadores.
17. O sistema deve suportar paginacao ou limitacao explicita da lista retornada.
18. O sistema deve comunicar claramente quando uma consulta nao retornar jogadores.
19. O sistema deve permitir que o site identifique o recorte aplicado em cada resposta.

## Experiência do Usuário

As personas centrais desta funcionalidade sao o visitante publico que visualiza o ranking no site e o time responsavel pelo frontend que consome a API.

Necessidades principais:

- visualizar rapidamente quem lidera o ranking
- confiar que a regra de ordenacao e consistente
- aplicar filtros compreensiveis para consultar recortes relevantes
- receber respostas previsiveis e simples de renderizar no site

Fluxos principais:

- o visitante acessa a pagina de ranking no site
- o site consulta a API publica com os filtros desejados
- a API retorna a lista ranqueada de jogadores elegiveis
- o site exibe a classificacao e o contexto da consulta

Consideracoes de UI e UX:

- a API deve favorecer nomenclaturas claras e consistentes para os filtros e os campos retornados
- a resposta deve permitir ao frontend explicar a ordenacao por eficiencia
- mensagens de ausencia de resultado devem ser simples de apresentar ao publico

Requisitos de acessibilidade:

- a API deve fornecer metadados e campos textuais claros para que o site possa construir uma exibicao acessivel
- a semantica dos dados retornados deve evitar abreviacoes ambiguas quando houver representacao publica

## Restrições Técnicas de Alto Nível

- A funcionalidade deve consumir a base analitica ja existente como fonte de dados do ranking.
- A funcionalidade nao deve alterar a responsabilidade dos jobs de coleta e consolidacao existentes.
- A funcionalidade deve servir o site existente por meio de contrato publico proprio, sem expor a estrutura interna do banco como interface oficial.
- A funcionalidade deve ocultar de forma padrao e permanente, nesta versao, partidas excluidas de analytics.
- A funcionalidade deve suportar volume de consultas compativel com consumo publico do site.
- Todo o codigo, nomes de arquivos, variaveis, schemas, endpoints e demais artefatos tecnicos associados devem estar em ingles.
- Esta funcionalidade deve considerar usabilidade e acessibilidade no desenho do contrato de resposta.

## Fora de Escopo

- construcao ou alteracao do frontend do site
- perfil detalhado de jogador
- historico de partidas por jogador
- dashboard geral ou KPIs agregados amplos
- autenticacao ou personalizacao por usuario
- comparacao entre jogadores
- painel administrativo
- definicao detalhada de arquitetura, schema, estrategia de consulta ou desenho de persistencia
