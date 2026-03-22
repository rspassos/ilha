# Template de Documento de Requisitos de Produto (PRD)

## Visão Geral

O `match-stats-collector` e um job interno responsavel por coletar dados dos ultimos jogos de multiplos servidores, consolidar as informacoes de resultados e estatisticas e persistir um historico confiavel para consumo posterior por outros componentes do repositorio.

O problema que esta funcionalidade resolve e a ausencia de uma base historica consistente para alimentar uma pagina de ranking por modo de jogo. Hoje os dados estao disponiveis apenas por meio de dois endpoints que retornam os ultimos 10 jogos, o que limita consultas futuras, dificulta consolidacao entre fontes e cria risco de perda de informacao caso a coleta nao seja feita continuamente.

O valor desta funcionalidade esta em criar uma camada confiavel de ingestao de dados, independente dos consumidores futuros, que permita evoluir ranking por modo de jogo e, mais adiante, estatisticas agregadas por jogador, sem depender diretamente dos endpoints operacionais em tempo de consulta.

## Objetivos

- Garantir a coleta recorrente de dados de partidas de uma lista configuravel de servidores.
- Consolidar, por partida, os dados vindos do endpoint de resultados e do endpoint de estatisticas.
- Persistir historico de partidas de forma confiavel para uso por consumidores futuros do repositorio.
- Disponibilizar uma base minima para alimentar ranking por modo de jogo como primeiro caso de uso de negocio.
- Possibilitar futura evolucao para estatisticas agregadas de jogadores sem exigir nova coleta dos dados originais.

Como e o sucesso:

- O job registra historico de partidas de todos os servidores configurados sem depender de consultas manuais.
- Os dados persistidos permitem identificar partidas por servidor, modo, mapa, data e participantes.
- Os dados persistidos sao suficientes para suportar a construcao futura de ranking por modo de jogo.
- O processo possui visibilidade basica de execucao, falhas e volume coletado.

Metricas principais para acompanhar:

- percentual de execucoes concluidas com sucesso por servidor
- quantidade de partidas novas persistidas por execucao
- quantidade de partidas consolidadas com dados de ambos os endpoints
- quantidade de falhas por endpoint e por servidor
- tempo medio de coleta por servidor

Objetivos de negocio:

- viabilizar uma pagina de ranking por modo de jogo com base historica propria
- reduzir dependencia de consultas online aos endpoints operacionais
- preparar a base para futuras estatisticas agregadas de jogadores

## Histórias de Usuário

- Como mantenedor da plataforma, eu quero coletar continuamente os dados de partidas de varios servidores para que o repositorio tenha um historico proprio e confiavel.
- Como futuro consumidor interno de ranking, eu quero acessar dados historicos consolidados por partida para que eu possa calcular rankings por modo de jogo sem depender dos endpoints de origem.
- Como futuro consumidor interno de analytics, eu quero que os dados de uma partida tragam resultado e estatisticas consolidadas para que eu possa derivar indicadores agregados de jogadores no futuro.
- Como operador do sistema, eu quero saber quando a coleta falha, quando um servidor nao responde e quantas partidas foram persistidas para que eu possa acompanhar a saude do job.
- Como mantenedor da plataforma, eu quero configurar quais servidores participam da coleta para que o processo possa crescer sem alterar seus objetivos de negocio.

Casos extremos relevantes:

- um servidor responde apenas um dos endpoints esperados
- os endpoints retornam estruturas diferentes dependendo do modo de jogo
- os mesmos jogos aparecem em multiplas execucoes consecutivas
- um servidor configurado fica temporariamente indisponivel

## Funcionalidades Principais

### 1. Coleta recorrente de partidas por servidor

A funcionalidade deve executar a coleta de dados de uma lista configuravel de servidores, buscando informacoes dos endpoints disponiveis para cada origem.

Por que e importante:

- sem coleta recorrente nao existe historico proprio
- os endpoints de origem possuem janela limitada aos ultimos 10 jogos

Como funciona em alto nivel:

- o job deve identificar os servidores habilitados para coleta
- para cada servidor, deve buscar os dados necessarios dos endpoints disponiveis

Requisitos funcionais:

1. O sistema deve permitir configurar quais servidores participam da coleta.
2. O sistema deve coletar dados de partidas de todos os servidores habilitados.
3. O sistema deve consumir os dois endpoints necessarios para obter resultados e estatisticas das partidas.
4. O sistema deve registrar a origem de cada partida coletada, incluindo o servidor de origem.

### 2. Consolidacao de dados de partida

A funcionalidade deve consolidar os dados provenientes dos dois endpoints em um unico registro logico por partida.

Por que e importante:

- resultados e estatisticas possuem valor limitado quando armazenados separadamente
- ranking e futuras agregacoes dependem de uma visao unica da partida

Como funciona em alto nivel:

- os dados retornados pelos endpoints devem ser correlacionados por partida
- o registro consolidado deve preservar os campos necessarios para consulta futura

Requisitos funcionais:

5. O sistema deve consolidar os dados de resultados e estatisticas referentes a uma mesma partida.
6. O sistema deve preservar os dados originais necessarios para futuras interpretacoes e evolucoes de consumo.
7. O sistema deve suportar partidas de diferentes modos de jogo, incluindo estruturas variaveis de dados.
8. O sistema deve registrar os atributos minimos que permitam identificar a partida, seu modo de jogo e o momento em que ocorreu.

### 3. Persistencia historica confiavel

A funcionalidade deve persistir o historico de partidas coletadas de forma que os dados possam ser consultados posteriormente por outros componentes internos.

Por que e importante:

- o caso de uso principal depende de uma base historica sob controle do projeto
- a coleta recorrente inevitavelmente reprocessara parte das mesmas partidas

Como funciona em alto nivel:

- as partidas consolidadas devem ser gravadas em armazenamento persistente
- o historico deve evitar duplicacoes indevidas quando uma mesma partida for coletada novamente

Requisitos funcionais:

9. O sistema deve persistir o historico de partidas coletadas.
10. O sistema deve evitar a criacao de registros duplicados para a mesma partida quando houver reprocessamento.
11. O sistema deve manter os dados persistidos disponiveis para consumidores internos futuros do repositorio.
12. O sistema deve armazenar dados suficientes para suportar ranking por modo de jogo como caso de uso inicial.

### 4. Observabilidade basica da coleta

A funcionalidade deve disponibilizar visibilidade operacional minima sobre o processo de coleta e persistencia.

Por que e importante:

- sem observabilidade, falhas silenciosas podem comprometer o historico
- o job rodara de forma isolada e independente, exigindo acompanhamento proprio

Como funciona em alto nivel:

- o processo deve emitir informacoes de sucesso, falha e volume de dados processados

Requisitos funcionais:

13. O sistema deve registrar o resultado de cada execucao de coleta por servidor.
14. O sistema deve registrar falhas de acesso aos endpoints e falhas de persistencia.
15. O sistema deve registrar a quantidade de partidas processadas e persistidas em cada execucao.
16. O sistema deve permitir identificar quando um servidor configurado nao pode ser coletado.

## Experiência do Usuário

Personas de usuario e suas necessidades:

- Mantenedor da plataforma: precisa de um processo previsivel, confiavel e observavel para formar a base historica.
- Consumidor interno de ranking: precisa que os dados persistidos sejam coerentes, historicos e organizados por modo de jogo.
- Consumidor interno de analytics futuro: precisa que o historico preserve informacoes suficientes para agregacoes posteriores por jogador.

Fluxos e interacoes principais do usuario:

- O mantenedor configura os servidores que devem ser coletados.
- O job executa a coleta de forma isolada e independente.
- O mantenedor acompanha se a execucao ocorreu com sucesso e se houve novas partidas persistidas.
- Componentes futuros do repositorio consomem a base persistida sem interagir diretamente com os endpoints de origem.

Consideracoes e requisitos de UI/UX:

- Esta funcionalidade nao possui interface grafica propria neste escopo.
- A experiencia principal do usuario e operacional, por meio de configuracao e observabilidade.
- Os nomes de arquivos, codigos, esquemas, variaveis e recursos associados devem estar em ingles.

Requisitos de acessibilidade:

- Nao se aplica diretamente neste ciclo, pois nao ha interface de usuario final.
- Eventuais saidas operacionais e documentacao associada devem priorizar clareza e consistencia terminologica.

## Restrições Técnicas de Alto Nível

- A funcionalidade deve operar a partir de dois endpoints externos existentes que retornam apenas os ultimos 10 jogos por servidor.
- A funcionalidade deve suportar payloads com variacao estrutural conforme o modo de jogo.
- A funcionalidade deve existir como componente isolado e independente dentro deste repositorio, com previsao de localizacao em `jobs/collector/`.
- A funcionalidade deve servir consumidores internos separados, sem acoplamento direto com API publica ou dashboard neste ciclo.
- Todo o codigo, nomes de arquivos, variaveis, schemas e demais artefatos tecnicos associados devem estar em ingles.
- O armazenamento deve preservar dados historicos de modo a permitir consumo futuro para ranking por modo de jogo e agregacoes posteriores.
- A funcionalidade deve possuir observabilidade basica suficiente para acompanhamento operacional.

## Fora de Escopo

- construcao de API de consulta
- construcao de dashboard ou pagina de ranking
- calculo e exposicao de ranking neste ciclo
- geracao de estatisticas agregadas de jogadores neste ciclo
- interfaces para usuarios finais
- definicao detalhada de arquitetura, schema, estrategia de execucao ou desenho de persistencia
- integracoes de consumo com outros componentes alem da persistencia do historico
