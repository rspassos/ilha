# Tarefa 2.0: Definir contrato HTTP público e validação de entrada do ranking

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Definir os tipos de requisição e resposta da API pública, a estratégia de validação dos parâmetros de consulta, os defaults da primeira versão e o formato de erro baseado em `application/problem+json`.

<skills>
### Conformidade com Skills Padrões

- `golang-patterns`: manter contratos pequenos, explícitos e fáceis de testar.
- `supabase-postgres-best-practices`: alinhar a validação de filtros com o que a consulta SQL suportará de forma eficiente.
</skills>

<requirements>
- Definir o contrato do endpoint `GET /v1/rankings/players`.
- Suportar query params para `mode`, `map`, `server`, `from`, `to`, `sort_by`, `sort_direction`, `limit` e `offset`.
- Aplicar defaults para ordenação, mínimo de partidas e paginação.
- Restringir `sort_by` à whitelist prevista na techspec.
- Padronizar erros de validação e erro interno via `problem+json`.
</requirements>

## Subtarefas

- [ ] 2.1 Definir tipos de `RankingQuery`, resposta JSON e metadados de paginação.
- [ ] 2.2 Implementar parser e validação dos query params do endpoint.
- [ ] 2.3 Implementar defaults de `sort_by`, `sort_direction`, `limit`, `offset` e `minimum_matches`.
- [ ] 2.4 Implementar writer de erros em `application/problem+json`.
- [ ] 2.5 Cobrir contratos e validações com testes automatizados.

## Detalhes de Implementação

Referenciar na `techspec.md`:

- `Design de Implementação > Interfaces Principais`
- `Design de Implementação > Modelos de Dados`
- `Design de Implementação > Endpoints de API`
- `Pontos de Integração > Tratamento de erros`

## Critérios de Sucesso

- O contrato HTTP fica definido de forma estável e consistente para o frontend.
- Parâmetros inválidos são rejeitados com resposta padronizada e clara.
- Defaults da API são aplicados de forma determinística.
- O contrato cobre integralmente o escopo de ranking aprovado no PRD.

## Testes da Tarefa

- [ ] Testes de unidade para parsing e validação de query params.
- [ ] Testes de unidade para defaults e whitelist de ordenação.
- [ ] Testes de unidade para serialização de `problem+json`.
- [ ] Testes de integração mínimos do contrato HTTP sem banco real, se existirem helpers de roteamento.
- [ ] Testes E2E (se aplicável): não se aplica nesta tarefa.

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `tasks/prd-exibicao-player-stats-api/prd.md`
- `tasks/prd-exibicao-player-stats-api/techspec.md`
- `services/player-stats-api/internal/httpapi`
- `services/player-stats-api/internal/service`
- `services/player-stats-api/internal/model`
