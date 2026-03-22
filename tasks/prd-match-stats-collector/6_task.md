# Tarefa 6.0: Orquestração do ciclo `run once` por servidor com tratamento parcial de falhas

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Integrar configuracao, cliente HTTP, merge e persistencia em um fluxo unico de execucao `run once`, pensado para ser disparado por scheduler externo. Esta tarefa entrega o comportamento funcional minimo do job de coleta.

<skills>
### Conformidade com Skills Padrões

Usar obrigatoriamente a skill [golang-patterns](/home/rspassos/projects/ilha/.agents/skills/golang-patterns/SKILL.md).
</skills>

<requirements>
- Implementar `RunOnce` para percorrer os servidores habilitados.
- Garantir isolamento de falha por servidor.
- Propagar contexto e erros de forma observavel.
- Persistir partidas consolidadas ao final de cada ciclo por servidor.
- Manter o binario compatível com execucao unica por scheduler externo.
- Aplicar `context.Context`, isolamento de responsabilidades e simplicidade de orquestracao conforme `golang-patterns`.
</requirements>

## Subtarefas

- [ ] 6.1 Implementar `CollectorService` e o fluxo por servidor.
- [ ] 6.2 Integrar fetch, merge e persistencia em uma unica execucao.
- [ ] 6.3 Implementar comportamento de falha parcial sem abortar servidores independentes.
- [ ] 6.4 Expor codigo de saida e resultado final adequados para uso por scheduler externo.

## Detalhes de Implementação

Referenciar:

- `Resumo Executivo`
- `Fluxo de dados`
- `Pontos de Integração`
- `Tratamento de erros`

em [techspec.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/techspec.md).

## Critérios de Sucesso

- O binario executa um ciclo completo de coleta e finaliza.
- Um servidor com erro nao impede a coleta dos demais.
- Partidas consolidadas sao persistidas ao final de cada servidor processado.
- O resultado da execucao e utilizavel por scheduler externo e por operacao manual.

## Testes da Tarefa

- [ ] Testes de unidade para fluxo por servidor e tratamento de falhas.
- [ ] Testes de integração cobrindo um ciclo completo com fixtures e PostgreSQL real.
- [ ] Testes E2E (se aplicável)

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `jobs/collector/cmd/collector/`
- `jobs/collector/internal/collector/`
- `jobs/collector/internal/httpclient/`
- `jobs/collector/internal/merge/`
- `jobs/collector/internal/storage/`
- [prd.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/prd.md)
- [techspec.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/techspec.md)
