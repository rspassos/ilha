# PRD: Consolidação de Player Stats

## Visão Geral

Hoje o projeto já coleta partidas e salva os payloads brutos em `collector_matches`. O próximo passo é transformar os dados de jogadores dessas partidas em uma base analítica própria, preparada para consumo interno futuro em dashboard e consultas analíticas.

O problema atual é que os dados por jogador estão embutidos em payloads brutos, o que dificulta filtragem, agregação histórica, comparação entre mapas e modos de jogo, e posterior vinculação de identidades de jogadores com múltiplos nomes.

Esta funcionalidade deve criar uma camada de dados consolidada por jogador e por partida, preservando rastreabilidade com os dados coletados e permitindo excluir jogos com bots no consumo analítico, sem perder o registro bruto.

## Objetivos

- Disponibilizar uma base analítica por jogador e por partida para consumo interno futuro.
- Permitir filtragem e agregação por jogador, modo, mapa, servidor e período.
- Garantir que jogos com bots permaneçam armazenados, mas possam ser excluídos facilmente do uso analítico.
- Criar uma base inicial de identidade de jogadores com jogador canônico e aliases observados.
- Preservar consistência e reprocessamento idempotente a partir da fonte de verdade existente.

Métricas de sucesso:

- 100% das partidas elegíveis em `collector_matches` geram registros analíticos por jogador.
- 100% dos registros analíticos possuem modo normalizado em um conjunto canônico.
- 100% dos nomes observados em partidas consolidadas ficam associados a um jogador canônico e a um alias.
- Jogos com bots podem ser removidos de consultas analíticas por filtro simples, sem reprocessamento manual.

## Histórias de Usuário

- Como pessoa de produto ou operação, eu quero consultar estatísticas históricas por jogador para entender desempenho ao longo do tempo.
- Como analista interno, eu quero filtrar dados por mapa, modo e período para comparar contextos de jogo diferentes.
- Como analista interno, eu quero excluir rapidamente partidas com bots das análises para evitar distorção nos indicadores.
- Como administrador futuro, eu quero que nomes observados sejam preservados como aliases para que eu possa vincular identidades depois sem perder histórico.
- Como time responsável pelo dashboard futuro, eu quero consumir dados já consolidados e normalizados para evitar depender de parsing de payload bruto em tempo de consulta.

Personas principais:

- Analista interno
- Time de produto
- Time responsável pelo dashboard analítico

Persona secundária:

- Administrador futuro de vinculação de aliases

## Funcionalidades Principais

### 1. Consolidação analítica de jogadores

A funcionalidade deve transformar cada partida coletada em registros analíticos por jogador, mantendo vínculo com a partida original.

Por que é importante:

- viabiliza consultas analíticas sem depender de leitura de JSON bruto
- reduz complexidade do dashboard futuro
- cria granularidade correta para métricas de performance individual

Requisitos funcionais:

1. O sistema deve consolidar dados por jogador a partir das partidas já registradas em `collector_matches`.
2. O sistema deve gerar um registro analítico para cada jogador presente em cada partida consolidada.
3. O sistema deve manter vínculo entre o registro analítico e a partida de origem.
4. O sistema deve preservar a informação bruta necessária para auditoria e reprocessamento futuro.

### 2. Normalização de modo de jogo

O modo precisa ser consistente para uso analítico, independentemente das diferenças entre fontes.

Por que é importante:

- garante comparabilidade entre partidas
- evita ambiguidade entre modos vindos de fontes diferentes

Requisitos funcionais:

5. O sistema deve persistir apenas os modos canônicos `1on1`, `2on2`, `3on3`, `4on4` e `dmm4`.
6. O sistema deve classificar a partida como `dmm4` quando a propriedade `dm` do `laststats` for igual a `4`.
7. O sistema deve classificar as demais partidas por quantidade de jogadores ativos, resultando em `1on1`, `2on2`, `3on3` ou `4on4`.
8. O sistema deve preservar os valores brutos de modo recebidos das fontes para auditoria.

### 3. Tratamento de partidas com bots

Jogos com bots não devem compor a análise padrão, mas precisam permanecer acessíveis.

Por que é importante:

- protege a qualidade analítica
- evita perda de histórico
- permite corrigir casos em que a origem não marca bots corretamente

Requisitos funcionais:

9. O sistema deve consolidar partidas com bots na mesma camada analítica das demais partidas.
10. O sistema deve marcar partidas com bots como excluídas do consumo analítico padrão.
11. O sistema deve preservar a indicação de presença de bots para permitir filtros e auditoria posteriores.

### 4. Identidade inicial de jogadores e aliases

Os nomes observados nas partidas devem formar uma base inicial de identidade, sabendo que a consolidação definitiva ocorrerá depois.

Por que é importante:

- prepara o terreno para consolidação administrativa futura
- evita bloquear a fase analítica por ausência de identidade definitiva

Requisitos funcionais:

12. O sistema deve criar automaticamente um jogador canônico quando encontrar um nome ainda não conhecido.
13. O sistema deve registrar o nome observado como alias vinculado ao jogador canônico correspondente.
14. O sistema deve reutilizar aliases já conhecidos quando o mesmo nome voltar a aparecer.
15. O sistema deve preservar o nome observado em cada registro analítico da partida.

### 5. Conjunto inicial de métricas analíticas

A primeira versão deve disponibilizar um conjunto de métricas suficiente para o dashboard inicial.

Por que é importante:

- atende o recorte analítico mais valioso sem inflar escopo
- mantém espaço para expansão futura

Requisitos funcionais:

16. O sistema deve disponibilizar, no mínimo, as métricas de eficiência, frags, deaths, kills, team kills, suicides, damage taken, damage given, spree max, spree quad, RL hits, RL kills, LG attacks, LG hits, percentual de acerto de LG, GA, RA, YA, `health_100`, ping, mapa e equipe do jogador.
17. O sistema deve calcular métricas derivadas necessárias ao consumo analítico, incluindo eficiência e percentual de acerto de LG.
18. O sistema deve permitir expansão futura do conjunto de métricas sem invalidar os registros já consolidados.

## Experiência do Usuário

O usuário desta funcionalidade não é o jogador final, e sim os times internos que consumirão os dados posteriormente. A experiência esperada é de confiança, consistência e simplicidade de consulta.

Necessidades principais:

- encontrar dados por jogador sem parsing manual
- comparar desempenho por modo e mapa
- excluir jogos com bots com um filtro simples
- confiar que aliases observados foram preservados

Considerações de UX e acessibilidade:

- o consumo futuro deve usar nomenclatura consistente para modos de jogo
- o status de exclusão analítica por bots deve ser explícito
- nomes observados e jogador canônico devem estar claramente distinguíveis no modelo de dados consumido
- o desenho do dado deve favorecer leitura simples por dashboard e consultas internas

## Restrições Técnicas de Alto Nível

- A fonte de verdade desta funcionalidade é a base já preenchida por `collector_matches`.
- Esta funcionalidade não pode alterar a responsabilidade atual do coletor de partidas.
- O processamento deve suportar reexecução segura sem duplicação lógica de registros.
- O modelo analítico deve preservar rastreabilidade com os dados brutos de origem.
- A solução deve tratar dados de jogadores como informação interna de uso operacional e analítico.
- A normalização de modos deve obedecer ao conjunto canônico definido neste PRD.

## Fora de Escopo

- Construção do dashboard analítico.
- Implementação do painel administrativo para consolidar aliases entre jogadores canônicos.
- Regras avançadas de reconciliação automática entre nomes diferentes do mesmo jogador.
- Novas coletas ou mudanças de responsabilidade no `jobs/collector`.
- Definição de visualizações, gráficos ou experiência final de BI.
- Expansão completa de todas as métricas disponíveis nos payloads além do conjunto inicial definido neste PRD.
