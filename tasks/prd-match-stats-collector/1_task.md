# Tarefa 1.0: Bootstrap do projeto `jobs/collector` e ambiente local de desenvolvimento

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Criar a estrutura inicial do projeto do coletor em Go e o ambiente de desenvolvimento local com Docker Compose, incluindo container do coletor, container PostgreSQL e arquivos base de configuracao local. O objetivo desta tarefa e deixar o projeto pronto para desenvolvimento incremental sem ainda implementar a logica de negocio completa.

<skills>
### Conformidade com Skills Padrões

Usar obrigatoriamente a skill [golang-patterns](/home/rspassos/projects/ilha/.agents/skills/golang-patterns/SKILL.md).
</skills>

<requirements>
- Criar a estrutura base em `jobs/collector/` conforme a tech spec.
- Adicionar arquivos de build e execucao local para o binario do coletor.
- Adicionar ambiente local com `docker-compose.yml` ou `compose.yml`.
- Incluir servicos `collector` e `postgres` na composicao local.
- Adicionar `.env.example` com placeholders para variaveis obrigatorias.
- Nao versionar segredos reais.
- Aplicar os principios da skill `golang-patterns` desde a estrutura inicial dos pacotes e entrypoints.
</requirements>

## Subtarefas

- [ ] 1.1 Criar a arvore inicial de diretorios e arquivos base do projeto Go em `jobs/collector/`.
- [ ] 1.2 Adicionar `Dockerfile` para build e execucao do coletor em ambiente local.
- [ ] 1.3 Adicionar `docker-compose.yml` com servicos `collector` e `postgres`.
- [ ] 1.4 Adicionar `.env.example` e estrategia de carregamento local de ambiente.
- [ ] 1.5 Garantir que o ambiente sobe localmente sem segredos reais embutidos.

## Detalhes de Implementação

Referenciar:

- `Resumo Executivo`
- `Arquitetura do Sistema`
- `Sequenciamento de Desenvolvimento`
- `Considerações Técnicas`

em [techspec.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/techspec.md).

## Critérios de Sucesso

- Existe uma estrutura inicial valida em `jobs/collector/`.
- O ambiente local define os dois containers previstos na tech spec.
- O projeto consegue iniciar localmente usando `.env` sem depender de credenciais embutidas no binario.
- Os arquivos base permitem a execucao das proximas tarefas sem retrabalho estrutural.

## Testes da Tarefa

- [ ] Testes de unidade para bootstrap minimo e validacao de configuracao inicial, se aplicavel.
- [ ] Testes de integração para subir o ambiente local e validar conectividade entre `collector` e `postgres`.
- [ ] Testes E2E (se aplicável)

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `jobs/collector/`
- `jobs/collector/Dockerfile`
- `compose.yml` ou `docker-compose.yml`
- `.env.example`
- [prd.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/prd.md)
- [techspec.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/techspec.md)
