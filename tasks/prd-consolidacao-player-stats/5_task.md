# Tarefa 5.0: Implementar resolução de identidade e aliases

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Implementar a camada de identidade inicial de jogadores, garantindo reaproveitamento por `login`, reaproveitamento por alias conhecido e criação automática de jogador canônico com alias quando necessário.

<skills>
### Conformidade com Skills Padrões

- `golang-patterns`: modelar a regra de resolução de identidade de forma explícita, testável e sem heurísticas implícitas.
- `supabase-postgres-best-practices`: garantir consistência entre resolução de identidade, aliases e persistência transacional.
</skills>

<requirements>
- Resolver o jogador canônico priorizando `login` quando disponível.
- Reutilizar alias conhecido quando não houver correspondência por `login`.
- Criar novo `player_canonical` e novo alias quando o jogador ainda não for conhecido.
- Atualizar janelas de observação dos aliases reaproveitados.
- Preservar sempre o nome observado no fato analítico da partida.
</requirements>

## Subtarefas

- [ ] 5.1 Definir modelos de entrada e saída da resolução de identidade.
- [ ] 5.2 Implementar busca prioritária por `login`.
- [ ] 5.3 Implementar fallback por alias conhecido.
- [ ] 5.4 Implementar criação de jogador canônico e alias novo quando necessário.
- [ ] 5.5 Cobrir cenários de reuso e criação com testes de unidade e integração.

## Detalhes de Implementação

Referenciar na `techspec.md`:

- `Arquitetura do Sistema > Fluxo de dados`
- `Design de Implementação > Interfaces Principais`
- `Design de Implementação > Modelos de Dados`
- `Abordagem de Testes > Testes Unidade`
- `Pontos de Integração > Tratamento de erros`

## Critérios de Sucesso

- Um mesmo `login` resolve sempre para o mesmo jogador canônico.
- Alias conhecido é reaproveitado quando não existir `login` resolvível.
- Jogadores novos são criados automaticamente sem intervenção manual.
- O histórico de nomes observados permanece preservado no fato e nos aliases.

## Testes da Tarefa

- [ ] Testes de unidade para reuso por `login`.
- [ ] Testes de unidade para reuso por alias.
- [ ] Testes de unidade para criação de novo jogador e novo alias.
- [ ] Testes de integração para evolução de alias com PostgreSQL real.
- [ ] Testes E2E (se aplicável): não se aplica nesta tarefa.

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `tasks/prd-consolidacao-player-stats/prd.md`
- `tasks/prd-consolidacao-player-stats/techspec.md`
- `jobs/player-stats/internal/identity`
- `jobs/player-stats/internal/storage`
- `jobs/player-stats/internal/model`
