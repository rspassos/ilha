# Tarefa 4.0: Consolidação de partidas por `demo` com metadados derivados

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Implementar a camada de merge que correlaciona `lastscores` e `laststats` pelo campo `demo`, gera um registro unico por partida e deriva os metadados necessarios para persistencia e filtros futuros. Esta tarefa entrega o nucleo de consolidacao do coletor.

<skills>
### Conformidade com Skills Padrões

Usar obrigatoriamente a skill [golang-patterns](/home/rspassos/projects/ilha/.agents/skills/golang-patterns/SKILL.md).
</skills>

<requirements>
- Correlacionar registros por `demo`.
- Gerar `match_key` com `server_key` e `demo_name`.
- Derivar metadados como `played_at`, `mode`, `map_name`, `participants` e `has_bots`.
- Preservar `score_payload`, `stats_payload` e `merged_payload`.
- Registrar warnings quando houver divergencia ou payload parcial.
- Manter a logica de merge simples, clara e testavel conforme `golang-patterns`.
</requirements>

## Subtarefas

- [ ] 4.1 Implementar indexacao por `demo` para ambos os conjuntos de dados.
- [ ] 4.2 Implementar regra de merge para partidas presentes em um ou nos dois endpoints.
- [ ] 4.3 Implementar derivacao de campos estruturados e da flag `has_bots`.
- [ ] 4.4 Implementar estrutura de warnings para correlacao parcial ou inconsistente.

## Detalhes de Implementação

Referenciar:

- `Fluxo de dados`
- `Modelos de Dados`
- `Regras de modelagem`
- `Riscos Conhecidos`

em [techspec.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/techspec.md).

## Critérios de Sucesso

- Cada `demo` resulta em no maximo um registro consolidado por servidor.
- O merge suporta payloads variaveis entre modos de jogo.
- A flag `has_bots` fica correta quando qualquer player indicar `is_bot=true`.
- Partidas parciais geram warning e continuam observaveis para diagnostico.

## Testes da Tarefa

- [ ] Testes de unidade para merge completo, merge parcial, deduplicacao e `has_bots`.
- [ ] Testes de integração com fixtures reais cobrindo `duel` e modos por equipe.
- [ ] Testes E2E (se aplicável)

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `jobs/collector/internal/merge/`
- `jobs/collector/internal/model/`
- [lastscores.json](/home/rspassos/projects/ilha/docs/responses/lastscores.json)
- [laststats.json](/home/rspassos/projects/ilha/docs/responses/laststats.json)
- [prd.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/prd.md)
- [techspec.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/techspec.md)
