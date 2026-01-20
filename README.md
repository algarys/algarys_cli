# Algarys CLI

CLI oficial da Algarys para criação e gerenciamento de projetos.

## Instalação

### Via script (recomendado)

```bash
curl -fsSL https://raw.githubusercontent.com/algarys/algarys/main/install.sh | bash
```

### Via Go

```bash
go install github.com/algarys/algarys@latest
```

### Manual

Baixe o binário da [página de releases](https://github.com/algarys/algarys/releases) e adicione ao seu PATH.

## Uso

### Criar novo projeto

```bash
algarys init
```

O comando interativo vai perguntar:
- Nome do projeto
- Descrição
- Versão do Python
- Se deseja criar repositório no GitHub

### Estrutura criada

```
meu-projeto/
├── meu_projeto/
│   ├── domain/           # Entidades e regras de negócio
│   ├── application/      # Casos de uso e serviços
│   ├── infrastructure/   # Implementações concretas
│   ├── interfaces/       # API e CLI
│   ├── ai/               # Agentes de IA
│   │   ├── agents/
│   │   ├── tools/
│   │   ├── prompts/
│   │   └── ...
│   └── temporal/         # Orquestração Temporal
│       ├── activities/
│       ├── workflows/
│       └── worker/
├── tests/
├── pyproject.toml
└── README.md
```

### Funcionalidades

- **Estrutura SOLID** - Arquitetura em camadas pronta para escalar
- **Módulo AI** - Estrutura para agentes, tools e prompts
- **Temporal** - Workers e workflows pré-configurados
- **GitHub** - Cria repo privado em `github.com/algarys` automaticamente
- **Ruleset** - Configura proteção da branch main (PR obrigatório + linear history)
- **UV** - Gerenciamento de dependências moderno

## Requisitos

Para usar todas as funcionalidades:

- [UV](https://docs.astral.sh/uv/) - Gerenciador de pacotes Python
- [GitHub CLI](https://cli.github.com/) - Para criar repositórios

```bash
# Instalar UV
curl -LsSf https://astral.sh/uv/install.sh | sh

# Instalar e autenticar GitHub CLI
brew install gh
gh auth login
```

## Desenvolvimento

### Build local

```bash
go build -o algarys .
```

### Criar release

```bash
git tag v0.1.0
git push origin v0.1.0
```

O GitHub Actions vai automaticamente criar os binários e a release.

---

Feito com Go + Cobra pela equipe Algarys
