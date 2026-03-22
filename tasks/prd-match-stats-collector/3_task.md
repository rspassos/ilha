# Tarefa 3.0: Cliente HTTP e parsing dos endpoints `lastscores` e `laststats`

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Implementar o cliente HTTP dos endpoints externos e o parsing dos payloads de `lastscores` e `laststats`, suportando variacoes estruturais entre modos de jogo. Esta tarefa entrega a capacidade de obter dados confiaveis da origem e transforma-los em estruturas internas validas.

<skills>
### Conformidade com Skills Padrões

Usar obrigatoriamente a skill [golang-patterns](/home/rspassos/projects/ilha/.agents/skills/golang-patterns/SKILL.md).
</skills>

<requirements>
- Consumir os dois endpoints definidos por servidor.
- Aplicar timeouts e tratamento de erros por request.
- Decodificar os payloads reais de `lastscores` e `laststats`.
- Preservar o campo `demo` para correlacao posterior.
- Suportar ao menos os modos de jogo cobertos pelas fixtures atuais.
- Propagar `context.Context` e envolver erros com contexto suficiente conforme `golang-patterns`.
</requirements>

## Subtarefas

- [ ] 3.1 Definir modelos internos de `ScoreMatch` e `StatsMatch`.
- [ ] 3.2 Implementar cliente HTTP reutilizavel com timeout configuravel.
- [ ] 3.3 Implementar decode dos endpoints com tratamento de erro e contexto.
- [ ] 3.4 Adicionar fixtures de teste baseadas nos exemplos reais em `docs/responses/`.

## Detalhes de Implementação

Referenciar:

- `Arquitetura do Sistema`
- `Pontos de Integração`
- `Abordagem de Testes`

em [techspec.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/techspec.md).

## Critérios de Sucesso

- O cliente consegue buscar ambos os endpoints de um servidor configurado.
- Os payloads reais de exemplo sao decodificados sem perda dos campos necessarios.
- Falhas HTTP e payloads invalidos retornam erros claros e contextuais.
- O resultado desta tarefa pode ser consumido pela camada de merge sem adaptacoes estruturais grandes.

## Testes da Tarefa

- [ ] Testes de unidade para decode de payload, timeouts e erro de transporte.
- [ ] Testes de integração com servidor HTTP de teste retornando fixtures reais.
- [ ] Testes E2E (se aplicável)

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `jobs/collector/internal/httpclient/`
- `jobs/collector/internal/model/`
- [lastscores.json](/home/rspassos/projects/ilha/docs/responses/lastscores.json)
- [laststats.json](/home/rspassos/projects/ilha/docs/responses/laststats.json)
- [prd.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/prd.md)
- [techspec.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/techspec.md)
