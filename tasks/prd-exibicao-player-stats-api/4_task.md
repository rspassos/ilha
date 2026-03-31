# Tarefa 4.0: Implementar serviço de domínio de ranking com filtros, ordenação e paginação

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Conectar o contrato de entrada da API à camada de storage por meio de um serviço de domínio que aplique defaults, normalize filtros, imponha regras de negócio da v1 e monte a resposta lógica do ranking.

<skills>
### Conformidade com Skills Padrões

- `golang-patterns`: orquestrar dependências por interfaces pequenas e manter a regra de negócio fácil de testar.
- `supabase-postgres-best-practices`: garantir que os filtros normalizados continuem compatíveis com a consulta agregada definida.
</skills>

<requirements>
- Implementar o `RankingService` com a operação de listagem pública.
- Aplicar defaults de ordenação, paginação e mínimo de partidas.
- Normalizar e propagar filtros aceitos ao repositório.
- Garantir que apenas campos de ordenação aprovados sejam usados.
- Preparar metadados da resposta de forma estável para o frontend.
</requirements>

## Subtarefas

- [ ] 4.1 Implementar o serviço de domínio de ranking e seu contrato principal.
- [ ] 4.2 Aplicar normalização de filtros e defaults da consulta.
- [ ] 4.3 Implementar validação de `sort_by` e `sort_direction` conforme whitelist e regras da v1.
- [ ] 4.4 Montar a resposta lógica com `data`, `meta` e `filters`.
- [ ] 4.5 Cobrir o serviço com testes de unidade usando mock do repositório.

## Detalhes de Implementação

Referenciar na `techspec.md`:

- `Design de Implementação > Interfaces Principais`
- `Design de Implementação > Modelos de Dados`
- `Arquitetura do Sistema > Fluxo de dados`
- `Abordagem de Testes > Testes Unidade`

## Critérios de Sucesso

- O serviço centraliza as regras de ranking da v1 sem duplicação nos handlers.
- Defaults e filtros são aplicados de forma consistente em todas as chamadas.
- O contrato lógico entregue ao handler é estável e pronto para serialização pública.
- Alterações futuras de filtro ou ordenação ficam confinadas à camada de serviço.

## Testes da Tarefa

- [ ] Testes de unidade para defaults de ordenação e paginação.
- [ ] Testes de unidade para normalização de filtros e rejeição de valores inválidos.
- [ ] Testes de unidade para montagem de `meta` e `filters`.
- [ ] Testes de integração leves com o repositório real, se necessários para fechar algum comportamento.
- [ ] Testes E2E (se aplicável): não se aplica nesta tarefa.

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `tasks/prd-exibicao-player-stats-api/prd.md`
- `tasks/prd-exibicao-player-stats-api/techspec.md`
- `services/player-stats-api/internal/service`
- `services/player-stats-api/internal/storage`
- `services/player-stats-api/internal/model`
