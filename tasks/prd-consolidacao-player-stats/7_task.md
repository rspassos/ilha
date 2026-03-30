# Tarefa 7.0: Fechar validação operacional local

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Completar a entrega com validação operacional local, integração ao fluxo de desenvolvimento do repositório e documentação suficiente para repetir a consolidação e verificar seus resultados com segurança.

<skills>
### Conformidade com Skills Padrões

- `golang-patterns`: manter a validação operacional alinhada à estrutura do projeto e ao fluxo padrão de execução.
- `supabase-postgres-best-practices`: validar o comportamento do job com PostgreSQL real e dados reaproveitados do ambiente local.
</skills>

<requirements>
- Incluir o novo job no fluxo local de validação do projeto.
- Atualizar `compose.yml` e/ou documentação operacional conforme necessário.
- Permitir execução repetida do job com comportamento idempotente observável.
- Validar cenários com bots, aliases e reprocessamento usando PostgreSQL real.
- Documentar comandos de execução e verificação de resultados consolidados.
</requirements>

## Subtarefas

- [X] 7.1 Integrar o job `player-stats` ao ambiente local de desenvolvimento.
- [X] 7.2 Definir ou atualizar comandos de execução e validação operacional.
- [X] 7.3 Validar reexecução idempotente e comportamento com partidas contendo bots.
- [X] 7.4 Validar evolução de aliases em ambiente realista.
- [X] 7.5 Consolidar documentação e testes finais da entrega.

## Detalhes de Implementação

Referenciar na `techspec.md`:

- `Sequenciamento de Desenvolvimento > Ordem de Construção`
- `Sequenciamento de Desenvolvimento > Dependências Técnicas`
- `Abordagem de Testes > Testes de Integração`
- `Monitoramento e Observabilidade`

## Critérios de Sucesso

- O time consegue subir o ambiente local e executar o job de consolidação sem fluxo ad hoc.
- A reexecução do job confirma comportamento idempotente.
- Partidas com bots permanecem consolidadas e filtráveis.
- A documentação operacional permite repetir a validação sem depender de conhecimento tácito.

## Testes da Tarefa

- [ ] Testes de unidade para helpers operacionais adicionados nesta etapa, se existirem.
- [ ] Testes de integração com PostgreSQL real e dados locais do projeto.
- [ ] Testes E2E locais do fluxo batch completo, incluindo reexecução idempotente.

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `tasks/prd-consolidacao-player-stats/prd.md`
- `tasks/prd-consolidacao-player-stats/techspec.md`
- `tasks/prd-consolidacao-player-stats/tasks.md`
- `compose.yml`
- `jobs/player-stats`
- `jobs/collector/config/collector.local.yaml`
