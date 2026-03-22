# Tarefa 7.0: Observabilidade com logs estruturados e métricas Prometheus

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Adicionar observabilidade basica ao coletor por meio de logs estruturados e metricas Prometheus locais. Esta tarefa entrega visibilidade operacional para execucao, falhas, volume processado e warnings de correlacao.

<skills>
### Conformidade com Skills Padrões

Usar obrigatoriamente a skill [golang-patterns](/home/rspassos/projects/ilha/.agents/skills/golang-patterns/SKILL.md).
</skills>

<requirements>
- Emitir logs estruturados para startup, execucao por servidor, warnings e falhas.
- Expor metricas Prometheus previstas na tech spec.
- Registrar latencia de requests e contagem de partidas processadas.
- Tornar visiveis warnings de merge e falhas de persistencia.
- Nao acoplar a feature a dashboards externos neste ciclo.
- Manter a instrumentacao idiomatica e sem abstrações desnecessarias, conforme `golang-patterns`.
</requirements>

## Subtarefas

- [ ] 7.1 Definir formato e campos obrigatorios dos logs estruturados.
- [ ] 7.2 Instrumentar contadores e histogramas Prometheus no fluxo principal.
- [ ] 7.3 Integrar emissao de warnings e erros com contexto de servidor e endpoint.
- [ ] 7.4 Validar exposicao local das metricas para uso em desenvolvimento e operacao.

## Detalhes de Implementação

Referenciar:

- `Monitoramento e Observabilidade`
- `Arquitetura do Sistema`
- `Tratamento de erros`

em [techspec.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/techspec.md).

## Critérios de Sucesso

- Existe visibilidade clara de inicio, fim, sucesso, warning e erro por servidor.
- As metricas previstas pela tech spec sao expostas localmente.
- O time operacional consegue diagnosticar mismatch, indisponibilidade e falhas de banco.
- A instrumentacao nao altera o comportamento funcional do coletor.

## Testes da Tarefa

- [ ] Testes de unidade para registro de metricas e formato dos eventos principais.
- [ ] Testes de integração validando exposicao de metricas e logs durante uma execucao real.
- [ ] Testes E2E (se aplicável)

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `jobs/collector/internal/metrics/`
- `jobs/collector/internal/collector/`
- `jobs/collector/cmd/collector/`
- [prd.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/prd.md)
- [techspec.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/techspec.md)
