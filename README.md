# Algarys CLI

CLI oficial da Algarys para criação e gerenciamento de projetos.

## Instalacao

**macOS / Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/algarys/algarys_cli/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/algarys/algarys_cli/main/install.ps1 | iex
```

### Outras opcoes

**Via Go (qualquer OS):**
```bash
go install github.com/algarys/algarys_cli@latest
```

**Manual:**
Baixe o binario da [pagina de releases](https://github.com/algarys/algarys_cli/releases) e adicione ao seu PATH.

## Comandos

### `algarys init`

Cria novo projeto Python com arquitetura SOLID.

```bash
algarys init
```

O comando interativo pergunta nome, descricao, versao do Python e se deseja criar repositorio no GitHub (requer login).

**Estrutura criada:**

```
meu-projeto/
├── meu_projeto/
│   ├── domain/           # Entidades e regras de negocio
│   ├── application/      # Casos de uso e servicos
│   ├── infrastructure/   # Implementacoes concretas
│   ├── interfaces/       # API e CLI
│   ├── ai/               # Agentes de IA
│   │   ├── agents/
│   │   ├── tools/
│   │   └── prompts/
│   └── temporal/         # Orquestracao Temporal
│       ├── activities/
│       ├── workflows/
│       └── worker/
├── tests/
├── pyproject.toml
└── README.md
```

### `algarys transcribe`

Transcreve arquivos de audio para texto usando OpenAI Whisper localmente.

```bash
# Passar caminho do arquivo
algarys transcribe gravacao.mp3

# Passar apenas o nome - busca automatica no computador
algarys transcribe reuniao.mp3

# Escolher modelo e idioma
algarys transcribe audio.wav -m tiny -l pt
```

**Flags:**
| Flag | Descricao | Default |
|------|-----------|---------|
| `-m, --model` | Modelo Whisper (tiny, base, small, medium, large) | large |
| `-l, --lang` | Codigo do idioma (pt, en, es...) | auto |

Na primeira execucao, o CLI configura automaticamente o ambiente Python com as dependencias necessarias.

**Requisitos:** [UV](https://docs.astral.sh/uv/) e [ffmpeg](https://ffmpeg.org/)

### `algarys login`

Autentica no GitHub para acessar funcionalidades da empresa.

```bash
algarys login
```

**Necessario para:** criar repositorios na org (`algarys init`), atualizar o CLI (`algarys update`).

**Nao necessario para:** transcrever audio (`algarys transcribe`).

### `algarys logout`

Desconecta da conta GitHub.

```bash
algarys logout
```

### `algarys update`

Atualiza o CLI para a ultima versao.

```bash
algarys update
```

### `algarys version`

Mostra a versao instalada.

```bash
algarys version
```

## Requisitos

| Ferramenta | Para que | Instalacao |
|------------|----------|------------|
| [UV](https://docs.astral.sh/uv/) | Projetos Python e transcricao | `curl -LsSf https://astral.sh/uv/install.sh \| sh` |
| [GitHub CLI](https://cli.github.com/) | Login e criar repos | `brew install gh` |
| [ffmpeg](https://ffmpeg.org/) | Transcricao de audio | `brew install ffmpeg` |

## Desenvolvimento

```bash
# Build local
go build -o algarys .

# Criar release (GitHub Actions gera os binarios)
git tag v0.1.0
git push origin v0.1.0
```

---

Feito com Go + Cobra pela equipe Algarys
