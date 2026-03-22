# Tarefa 5.0: Persistência idempotente em PostgreSQL com schema e `UPSERT`

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Implementar o schema inicial de banco e a camada de persistencia idempotente do coletor. Esta tarefa entrega armazenamento historico confiavel, com campos indexaveis e preservacao dos payloads JSON necessarios para consumidores futuros.

<skills>
### Conformidade com Skills Padrões

Usar obrigatoriamente a skill [golang-patterns](/home/rspassos/projects/ilha/.agents/skills/golang-patterns/SKILL.md).
</skills>

<requirements>
- Criar schema inicial para `collector_matches`.
- Criar constraint unica em `server_key` e `demo_name`.
- Persistir payloads bruto e merged em `JSONB`.
- Persistir a coluna `has_bots` para filtro rapido.
- Garantir `UPSERT` idempotente para reprocessamento de partidas.
- Seguir interfaces pequenas e erros com contexto conforme `golang-patterns`.
</requirements>

## Subtarefas

- [ ] 5.1 Criar migracao inicial do banco para a tabela `collector_matches`.
- [ ] 5.2 Implementar repositorio PostgreSQL para insert e update idempotentes.
- [ ] 5.3 Implementar mapeamento entre `MatchRecord` e schema persistido.
- [ ] 5.4 Validar indices e colunas essenciais para filtros do caso de uso inicial.

## Detalhes de Implementação

Referenciar:

- `Modelos de Dados`
- `Schema de banco sugerido`
- `constraints e indices`
- `Decisões Principais`

em [techspec.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/techspec.md).

## Critérios de Sucesso

- O banco possui schema inicial alinhado com a tech spec.
- A mesma partida nao gera duplicidade em reprocessamentos.
- Os campos estruturados e os payloads `JSONB` ficam persistidos corretamente.
- O filtro por `has_bots` e os filtros por `mode` e `server_key` sao suportados pelo schema.

## Testes da Tarefa

- [ ] Testes de unidade para mapeamento de persistencia e regras de `UPSERT`.
- [ ] Testes de integração contra PostgreSQL validando migracao, persistencia e idempotencia.
- [ ] Testes E2E (se aplicável)

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `jobs/collector/internal/storage/`
- diretorio de migracoes do coletor
- `compose.yml` ou `docker-compose.yml`
- [prd.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/prd.md)
- [techspec.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/techspec.md)
