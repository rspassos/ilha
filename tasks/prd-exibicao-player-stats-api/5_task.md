# Tarefa 5.0: Expor o endpoint `GET /v1/rankings/players` com respostas JSON e `problem+json`

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Implementar a camada HTTP pública da funcionalidade, conectando roteamento, handlers, serialização JSON, respostas de erro e integração com o serviço de domínio do ranking.

<skills>
### Conformidade com Skills Padrões

- `golang-patterns`: manter handlers finos, sem lógica de negócio e com tratamento explícito de erros.
- `supabase-postgres-best-practices`: preservar compatibilidade entre filtros recebidos via HTTP e consulta real já suportada no storage.
</skills>

<requirements>
- Expor publicamente o endpoint `GET /v1/rankings/players`.
- Integrar o handler ao `RankingService`.
- Retornar JSON estável com `data`, `meta` e `filters`.
- Retornar erros de validação e falha interna em `application/problem+json`.
- Garantir status codes coerentes e mensagens adequadas para consumo do site.
</requirements>

## Subtarefas

- [ ] 5.1 Implementar roteamento HTTP do endpoint de ranking.
- [ ] 5.2 Implementar handler integrado ao serviço de domínio.
- [ ] 5.3 Implementar serialização da resposta pública e cabeçalhos relevantes.
- [ ] 5.4 Integrar `problem+json` aos fluxos de erro do endpoint.
- [ ] 5.5 Cobrir o endpoint com testes HTTP automatizados.

## Detalhes de Implementação

Referenciar na `techspec.md`:

- `Arquitetura do Sistema > Visão Geral dos Componentes`
- `Design de Implementação > Endpoints de API`
- `Pontos de Integração > Tratamento de erros`
- `Abordagem de Testes > Testes Unidade`

## Critérios de Sucesso

- O endpoint público responde corretamente para consultas válidas.
- O frontend passa a ter um contrato HTTP completo para consumir ranking.
- Erros de entrada e falhas internas ficam padronizados e previsíveis.
- O handler permanece fino e sem acoplamento indevido à camada SQL.

## Testes da Tarefa

- [ ] Testes de unidade dos handlers com mock do serviço.
- [ ] Testes de unidade para respostas `problem+json`.
- [ ] Testes de integração HTTP cobrindo sucesso, consulta vazia e parâmetros inválidos.
- [ ] Testes de integração HTTP cobrindo ordenação e paginação básicas.
- [ ] Testes E2E (se aplicável): não se aplica nesta tarefa.

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `tasks/prd-exibicao-player-stats-api/prd.md`
- `tasks/prd-exibicao-player-stats-api/techspec.md`
- `services/player-stats-api/internal/httpapi`
- `services/player-stats-api/internal/service`
- `services/player-stats-api/internal/model`
- `services/player-stats-api/cmd/player-stats-api`
