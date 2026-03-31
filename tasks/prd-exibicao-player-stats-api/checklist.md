# Checklist Final — Tarefa 7.0

Status: Em progresso

- [ ] Subir Postgres local para testes de integração (`docker compose up -d postgres`).
- [ ] Executar testes de integração em `services/player-stats-api` com `PLAYER_STATS_API_TEST_DATABASE_URL` apontando para o DB.
- [ ] Validar cenários combinados: filtros por `mode`, `map`, `server`, intervalo `from/to`.
- [ ] Validar ordenação suportada (`efficiency`, `frags`, `lg_accuracy`, etc.) e direção (`asc`, `desc`).
- [ ] Validar aplicação do `minimum_matches` (padrão 10) e exclusão de partidas inelegíveis (`excluded_from_analytics` / `excluded_from_rank`).
- [ ] Validar paginação e ordem determinística quando `limit`/`offset` variam.
- [ ] Produzir exemplos finais de chamadas HTTP e payloads de resposta em `techspec.md`.
- [ ] Atualizar `tasks/prd-exibicao-player-stats-api/validation_report.md` com resultados e logs passados.
- [ ] Adicionar/rodar testes unitários adicionais se algum helper for adicionado nesta etapa.

Comandos úteis:

1. Subir o stack mínimo (apenas Postgres):

```bash
docker compose -f compose.yml up -d postgres
```

2. Executar testes (dentro do repositório):

```bash
cd services/player-stats-api
PLAYER_STATS_API_TEST_DATABASE_URL='postgres://ilha:ilha@127.0.0.1:5432/ilha?sslmode=disable' \
  GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod-cache go test ./... -v
```

3. Se os testes travarem por lock advisory, verifique conexões ativas e reinicie o DB:

```bash
docker compose restart postgres
```
