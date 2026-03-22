# Tarefa 2.0: Configuração do coletor via YAML e variáveis de ambiente

<critical>Ler os arquivos de prd.md e techspec.md desta pasta, se você não ler esses arquivos sua tarefa será invalidada</critical>

## Visão Geral

Implementar o carregamento e a validacao da configuracao do coletor a partir de arquivo YAML e variaveis de ambiente. Esta tarefa estabelece a base para identificar servidores habilitados, parametros globais e configuracoes obrigatorias de conexao.

<skills>
### Conformidade com Skills Padrões

Usar obrigatoriamente a skill [golang-patterns](/home/rspassos/projects/ilha/.agents/skills/golang-patterns/SKILL.md).
</skills>

<requirements>
- Ler configuracao dos servidores por arquivo YAML.
- Ler variaveis sensiveis e de conexao por ambiente.
- Validar campos obrigatorios no startup.
- Produzir erros claros para configuracoes invalidas ou incompletas.
- Manter todo o codigo e nomenclatura tecnica em ingles.
- Aplicar tratamento idiomatico de erros e design simples de tipos conforme `golang-patterns`.
</requirements>

## Subtarefas

- [ ] 2.1 Definir structs de configuracao e contrato esperado do YAML.
- [ ] 2.2 Implementar loader de configuracao de arquivo e ambiente.
- [ ] 2.3 Implementar validacao de campos obrigatorios e defaults seguros.
- [ ] 2.4 Documentar o contrato minimo de configuracao para uso local.

## Detalhes de Implementação

Referenciar:

- `Arquitetura do Sistema`
- `Modelos de Dados`
- `Configuracao por ambiente`
- `Riscos Conhecidos`

em [techspec.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/techspec.md).

## Critérios de Sucesso

- O coletor consegue carregar uma lista de servidores habilitados via YAML.
- Variaveis obrigatorias, como `DATABASE_URL`, sao lidas do ambiente.
- Configuracoes invalidas falham cedo com mensagem clara.
- O comportamento e previsivel em ambiente local e futuro ambiente de execucao.

## Testes da Tarefa

- [ ] Testes de unidade para parsing de YAML, defaults e validacoes.
- [ ] Testes de integração carregando configuracao realista com `.env` e arquivo YAML.
- [ ] Testes E2E (se aplicável)

<critical>SEMPRE CRIE E EXECUTE OS TESTES DA TAREFA ANTES DE CONSIDERÁ-LA FINALIZADA</critical>

## Arquivos relevantes

- `jobs/collector/internal/config/`
- `.env.example`
- arquivo YAML de configuracao local
- [prd.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/prd.md)
- [techspec.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/techspec.md)
