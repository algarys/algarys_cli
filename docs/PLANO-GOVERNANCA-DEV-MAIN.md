# Plano de governança `dev` → `main`

## Objetivo

Fazer com que novos projetos criados pelo `algarys init` já nasçam com o fluxo de desenvolvimento:

```text
feature/* → dev → main
```

A branch `dev` será a integração do projeto. A branch `main` será a fonte de verdade e de release.

## Diagnóstico atual

O repositório `algarys_cli` atualmente:

- possui apenas a branch `main`;
- não possui a branch `dev`;
- possui apenas o ruleset organizacional de nome de repositório;
- inicializa e publica novos projetos somente na branch `main`;
- configura atualmente um ruleset para `main` exigindo PR, uma aprovação e histórico linear;
- possui workflow apenas para gerar releases a partir de tags `v*`.

Como referência, `algarys_escrever-historias` e `algarys_cachos_brasil` utilizam o fluxo `feature/* → dev → main`.

## Política desejada

### `dev`

Ruleset `Protect dev`:

- exige Pull Request;
- exige pelo menos uma aprovação;
- não possui bypass actors;
- permite merge, squash ou rebase.

### `main`

Ruleset `Protect main`:

- exige Pull Request;
- não exige aprovação para PRs de `dev`;
- exige o check do workflow que valida a origem do PR;
- bloqueia qualquer origem diferente de `dev` ou `hotfix/*` do próprio repositório;
- mantém o histórico linear atualmente utilizado;
- não possui bypass actors.

O GitHub Ruleset não consegue validar nativamente a branch de origem de um PR. Por isso, a restrição `dev → main` será implementada por um check obrigatório do GitHub Actions.

## Bootstrap do próprio `algarys_cli`

Como o CLI ainda não possui `dev`, a configuração deverá ocorrer nesta ordem:

1. Criar a branch remota `dev` a partir de `main`.
2. Adicionar diretamente em `dev` o workflow de validação da origem para `main`.
3. Abrir e mergear o PR `dev → main` para levar o workflow à `main`.
4. Criar o ruleset `Protect dev`.
5. Ativar/configurar o ruleset `Protect main` com aprovação zero e o check obrigatório.

O ruleset de `main` só deverá ser ativado depois que o workflow existir na própria `main`, evitando bloquear o PR de bootstrap.

## Workflow de validação

Arquivo planejado:

```text
.github/workflows/enforce-dev-source-to-main.yml
```

O workflow deverá:

- executar em `pull_request_target` direcionado à `main`;
- não fazer checkout do código da branch de origem;
- não acessar secrets;
- validar `base.ref == "main"`;
- validar `head.ref == "dev"`;
- validar que a branch `dev` pertence ao próprio repositório;
- falhar para qualquer outra origem.

Nome planejado do check:

```text
Enforce dev as main source / only-dev-source
```

## Alterações no `algarys init`

O gerador deverá:

- gerar o workflow de validação em todo projeto novo;
- inicializar o projeto com `main` e `dev` apontando para o commit inicial;
- publicar as duas branches no GitHub;
- configurar automaticamente os rulesets de `dev` e `main` após o push;
- manter o padrão de nome `algarys_<projeto>`;
- adicionar ao README gerado o fluxo `feature/* → dev → main`;
- gerar um template de Pull Request com descrição, motivo, testes e checklist;
- substituir `configureRuleset` por uma configuração dos dois rulesets;
- tornar as chamadas ao `gh` testáveis e reportar erros detalhados.

Projetos criados sem repositório remoto ainda deverão receber localmente as branches `main` e `dev` e os arquivos de governança.

## Fluxo de inserção do novo código no GitHub

Depois do bootstrap do CLI:

```text
feature/init-dev-main-governance → dev
                                  ↓
                              dev → main
                                  ↓
                              tag v0.9.0
```

- PR de feature para `dev`: requer aprovação.
- PR de `dev` para `main`: não requer aprovação, mas exige o workflow.
- Tag `v0.9.0`: aciona o workflow atual de release multiplataforma.

## Testes planejados

Adicionar testes para:

- criação local das branches `main` e `dev`;
- geração do workflow;
- conteúdo e payload dos rulesets;
- ordem dos comandos `gh`;
- falhas de autenticação, push e API;
- geração do README e template de PR.

Validação final:

```bash
go test ./...
go vet ./...
gofmt -l .
go build ./...
```

## Status da implementação local

A implementação local foi realizada no `algarys_cli`:

- o `algarys init` gera as branches `main` e `dev` no commit inicial;
- o gerador cria o workflow de validação da origem e o template de PR;
- repositórios remotos recebem as duas branches;
- os rulesets `Protect dev` e `Protect main` são criados automaticamente;
- o check obrigatório `Enforce dev as main source / only-dev-source` protege `main`;
- falhas de git, `gh`, autenticação, push e API são retornadas com contexto;
- testes cobrem arquivos gerados, branches, payloads, ordem dos comandos e falhas.

O próprio repositório `algarys_cli` permanece fora dessa governança por
orientação do proprietário: usa somente `main`, sem branch `dev`, PR obrigatório
ou rulesets de proteção. A governança é aplicada aos projetos novos gerados pelo
CLI. O commit e o push desta implementação serão feitos diretamente em `main`.
