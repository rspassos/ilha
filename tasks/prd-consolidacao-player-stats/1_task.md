# Tarefa 1.0: Estruturar o job batch `player-stats`

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Criar o esqueleto funcional do novo job `jobs/player-stats`, seguindo os padrões operacionais já usados no collector para configuração, bootstrap, logging, métricas e execução batch simples.

<skills>
### Conformidade com Skills Padrões

- `golang-patterns`: aplicar organização idiomática de packages, boundary interfaces e tratamento explícito de erros.
- `supabase-postgres-best-practices`: considerar desde o início a integração limpa com PostgreSQL e o desenho do bootstrap para migrações e conexões.
</skills>

<requirements>
- Criar o binário `jobs/player-stats/cmd/player-stats`.
- Estruturar packages iniciais de `config`, `bootstrap` e `metrics`.
- Reutilizar o padrão operacional do collector sem alterar a responsabilidade do `jobs/collector`.
- Permitir inicialização controlada do job com carregamento de configuração e conexão com PostgreSQL.
- Garantir que o job falhe com mensagens claras em caso de configuração inválida ou dependência indisponível.
</requirements>

## Subtarefas

- [ ] 1.1 Criar a estrutura inicial de diretórios e entrypoint do novo job.
- [ ] 1.2 Implementar carga de configuração e validações básicas do runtime.
- [ ] 1.3 Implementar bootstrap com logger, conexão PostgreSQL e inicialização de métricas.
- [ ] 1.4 Garantir execução batch simples e encerramento limpo do processo.
- [ ] 1.5 Cobrir bootstrap e configuração com testes automatizados.

## Detalhes de Implementação

Referenciar na `techspec.md`:

- `Arquitetura do Sistema > Visão Geral dos Componentes`
- `Design de Implementação > Interfaces Principais`
- `Sequenciamento de Desenvolvimento > Ordem de Construção`
- `Monitoramento e Observabilidade`

## Critérios de Sucesso

- O binário `player-stats` inicia corretamente com configuração válida.
- O bootstrap conecta logger, métricas e PostgreSQL sem depender ainda da consolidação completa.
- A estrutura criada suporta a evolução das próximas tarefas sem retrabalho arquitetural relevante.
- Falhas de configuração ou conexão retornam erro explícito e rastreável.

## Testes da Tarefa

- [ ] Testes de unidade para parsing e validação de configuração.
- [ ] Testes de unidade para bootstrap e tratamento de erros de inicialização.
- [ ] Testes de integração mínimos para subir o job com PostgreSQL disponível.
- [ ] Testes E2E (se aplicável): não se aplica nesta tarefa.

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `tasks/prd-consolidacao-player-stats/prd.md`
- `tasks/prd-consolidacao-player-stats/techspec.md`
- `jobs/player-stats/cmd/player-stats`
- `jobs/player-stats/internal/config`
- `jobs/player-stats/internal/bootstrap`
- `jobs/player-stats/internal/metrics`
- `jobs/collector/cmd/collector/main.go`
- `jobs/collector/internal/bootstrap/bootstrap.go`
- `jobs/collector/internal/config/config.go`
- `jobs/collector/internal/metrics/metrics.go`
