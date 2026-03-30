# Tarefa 4.0: Implementar normalização e extração de métricas por jogador

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Transformar o `stats_payload` de cada partida em registros analíticos por jogador, com modo normalizado, métricas base, métricas derivadas, snapshot bruto do jogador e flags analíticas relacionadas a bots.

<skills>
### Conformidade com Skills Padrões

- `golang-patterns`: manter a lógica de transformação pura e fácil de testar.
- `supabase-postgres-best-practices`: alinhar o formato produzido com o schema analítico e suas constraints.
</skills>

<requirements>
- Extrair um registro analítico para cada jogador consolidável da partida.
- Normalizar o modo para `1on1`, `2on2`, `3on3`, `4on4` ou `dmm4`.
- Calcular `efficiency` e `lg_accuracy` com tratamento de divisão por zero.
- Preservar `raw_mode`, nome observado, login observado, time, mapa e snapshot bruto do jogador.
- Marcar partidas com bots com `excluded_from_analytics=true` sem descartar a consolidação.
</requirements>

## Subtarefas

- [ ] 4.1 Implementar parser do recorte de jogador a partir de `stats_payload`.
- [ ] 4.2 Implementar normalização do modo canônico conforme `dm` e quantidade de jogadores ativos.
- [ ] 4.3 Implementar extração das métricas base previstas no PRD.
- [ ] 4.4 Implementar cálculos derivados e flags analíticas de bots.
- [ ] 4.5 Cobrir transformação e cálculos com testes unitários e integração com fixtures.

## Detalhes de Implementação

Referenciar na `techspec.md`:

- `Arquitetura do Sistema > Fluxo de dados`
- `Design de Implementação > Modelos de Dados`
- `Abordagem de Testes > Testes Unidade`
- `Considerações Técnicas > Riscos Conhecidos`

## Critérios de Sucesso

- Cada partida elegível gera registros por jogador com o conjunto mínimo de métricas do PRD.
- O modo persistido é sempre um valor canônico permitido.
- `efficiency` e `lg_accuracy` são calculados corretamente, inclusive em casos limite.
- Jogos com bots continuam consolidados, mas já saem prontos para exclusão analítica por filtro.

## Testes da Tarefa

- [ ] Testes de unidade para normalização de modo, incluindo `dm=4`.
- [ ] Testes de unidade para extração de métricas e cálculos derivados.
- [ ] Testes de unidade para regras de bots e snapshot bruto do jogador.
- [ ] Testes de integração com fixtures reais de `stats_payload`.
- [ ] Testes E2E (se aplicável): não se aplica nesta tarefa.

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `tasks/prd-consolidacao-player-stats/prd.md`
- `tasks/prd-consolidacao-player-stats/techspec.md`
- `jobs/player-stats/internal/normalize`
- `jobs/player-stats/internal/model`
- `jobs/player-stats/internal/source`
- `jobs/collector/internal/model/match.go`
