# Tarefa 6.0: Integrar observabilidade e operação local do serviço

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Completar a operação do novo serviço com métricas Prometheus, logs estruturados, integração ao `compose.yml` e comandos locais suficientes para execução e verificação no ambiente do projeto.

<skills>
### Conformidade com Skills Padrões

- `golang-patterns`: manter a operação alinhada ao padrão já usado no repositório para serviços Go.
- `supabase-postgres-best-practices`: validar o comportamento do serviço com PostgreSQL real e filtros representativos.
</skills>

<requirements>
- Expor métricas Prometheus do serviço HTTP.
- Registrar logs estruturados de startup, request, validação e falha interna.
- Integrar o novo serviço ao fluxo local de desenvolvimento em `compose.yml`.
- Permitir execução repetida do serviço em ambiente local sem configuração ad hoc.
- Documentar comandos operacionais mínimos necessários para desenvolvimento e validação.
</requirements>

## Subtarefas

- [ ] 6.1 Implementar coletores Prometheus e middleware de instrumentação do endpoint.
- [ ] 6.2 Implementar logs estruturados para lifecycle do serviço e requests HTTP.
- [ ] 6.3 Integrar o serviço ao `compose.yml` e ao ambiente local.
- [ ] 6.4 Definir ou atualizar comandos de execução e inspeção operacional.
- [ ] 6.5 Cobrir observabilidade e operação básica com testes automatizados e validação local.

## Detalhes de Implementação

Referenciar na `techspec.md`:

- `Monitoramento e Observabilidade`
- `Sequenciamento de Desenvolvimento > Dependências Técnicas`
- `Arquitetura do Sistema > Visão Geral dos Componentes`
- `Considerações Técnicas > Decisões Principais`

## Critérios de Sucesso

- O serviço expõe métricas Prometheus compatíveis com o padrão do repositório.
- Logs estruturados permitem rastrear startup, requests e falhas principais.
- O time consegue subir o serviço localmente junto com PostgreSQL e o schema analítico existente.
- A documentação operacional mínima permite repetir a validação sem conhecimento tácito.

## Testes da Tarefa

- [ ] Testes de unidade para coletores e helpers de observabilidade, se existirem.
- [ ] Testes de integração para endpoint de métricas e logs gerados em fluxos principais.
- [ ] Testes de integração locais com `compose` e PostgreSQL real, quando aplicável.
- [ ] Testes E2E (se aplicável): não se aplica nesta tarefa.

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `tasks/prd-exibicao-player-stats-api/prd.md`
- `tasks/prd-exibicao-player-stats-api/techspec.md`
- `tasks/prd-exibicao-player-stats-api/tasks.md`
- `services/player-stats-api/internal/metrics`
- `services/player-stats-api/internal/httpapi`
- `services/player-stats-api/internal/bootstrap`
- `compose.yml`
- `README.md`
