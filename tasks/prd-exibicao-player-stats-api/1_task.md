# Tarefa 1.0: Estruturar o novo serviço `player-stats-api`

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Criar o esqueleto funcional do novo serviço HTTP `player-stats-api`, seguindo os padrões já usados no repositório para configuração, bootstrap, logging, métricas, conexão com PostgreSQL e ciclo de vida do processo.

<skills>
### Conformidade com Skills Padrões

- `golang-patterns`: aplicar organização idiomática de packages, separação clara de responsabilidades e tratamento explícito de erros.
- `supabase-postgres-best-practices`: manter integração limpa com PostgreSQL desde o bootstrap do serviço.
</skills>

<requirements>
- Criar o binário `services/player-stats-api/cmd/player-stats-api`.
- Estruturar packages iniciais de `config`, `bootstrap`, `metrics` e `logging`.
- Reutilizar o padrão operacional dos serviços batch existentes sem alterar `jobs/collector` ou `jobs/player-stats`.
- Permitir inicialização controlada com carregamento de configuração e conexão com PostgreSQL.
- Garantir encerramento limpo dos servidores HTTP e do pool de banco.
</requirements>

## Subtarefas

- [ ] 1.1 Criar a estrutura inicial de diretórios e entrypoint do novo serviço.
- [ ] 1.2 Implementar carga de configuração e validações básicas do runtime.
- [ ] 1.3 Implementar bootstrap com logger, pool PostgreSQL e servidor de métricas.
- [ ] 1.4 Garantir startup e shutdown limpos do processo HTTP.
- [ ] 1.5 Cobrir bootstrap e configuração com testes automatizados.

## Detalhes de Implementação

Referenciar na `techspec.md`:

- `Arquitetura do Sistema > Visão Geral dos Componentes`
- `Sequenciamento de Desenvolvimento > Ordem de Construção`
- `Sequenciamento de Desenvolvimento > Dependências Técnicas`
- `Monitoramento e Observabilidade`

## Critérios de Sucesso

- O binário `player-stats-api` inicia corretamente com configuração válida.
- O bootstrap conecta logger, métricas e PostgreSQL sem depender ainda do endpoint final completo.
- A estrutura criada suporta as próximas tarefas sem retrabalho arquitetural relevante.
- Falhas de configuração ou conexão retornam erros claros e rastreáveis.

## Testes da Tarefa

- [ ] Testes de unidade para parsing e validação de configuração.
- [ ] Testes de unidade para bootstrap e tratamento de erros de inicialização.
- [ ] Testes de integração mínimos para subir o serviço com PostgreSQL disponível.
- [ ] Testes E2E (se aplicável): não se aplica nesta tarefa.

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `tasks/prd-exibicao-player-stats-api/prd.md`
- `tasks/prd-exibicao-player-stats-api/techspec.md`
- `services/player-stats-api/cmd/player-stats-api`
- `services/player-stats-api/internal/config`
- `services/player-stats-api/internal/bootstrap`
- `services/player-stats-api/internal/metrics`
- `services/player-stats-api/internal/logging`
- `jobs/player-stats/internal/bootstrap/bootstrap.go`
- `jobs/player-stats/internal/logging/logger.go`
- `compose.yml`
