# Resumo de Tarefas de Implementação de Match Stats Collector

## Tarefas

- [x] 1.0 Bootstrap do projeto `jobs/collector` e ambiente local de desenvolvimento
- [x] 2.0 Configuração do coletor via YAML e variáveis de ambiente
- [x] 3.0 Cliente HTTP e parsing dos endpoints `lastscores` e `laststats`
- [x] 4.0 Consolidação de partidas por `demo` com metadados derivados
- [x] 5.0 Persistência idempotente em PostgreSQL com schema e `UPSERT`
- [x] 6.0 Orquestração do ciclo `run once` por servidor com tratamento parcial de falhas
- [x] 7.0 Observabilidade com logs estruturados e métricas Prometheus
- [x] 8.0 Documentação operacional e validação do fluxo local integrado
