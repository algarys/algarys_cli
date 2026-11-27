package templates

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Fakes do docker e do gh: o registro é um arquivo "ref digest" por linha e
// toda chamada vai para um log, para o teste ver o que foi (ou não) reetiquetado.
const fakeDocker = `#!/bin/bash
echo "$*" >> "$FAKE_DIR/docker.log"
reg="$FAKE_DIR/registro"
touch "$reg"
digest_de() { awk -v r="$1" '$1 == r { print $2 }' "$reg"; }
case "$3" in
  inspect)
    d=$(digest_de "$4")
    [[ -n "$d" ]] || { echo "ERROR: $4: not found" >&2; exit 1; }
    printf '{"digest":"%s"}\n' "$d"
    ;;
  create)
    d=$(digest_de "$6")
    [[ -n "$d" ]] || { echo "ERROR: $6: not found" >&2; exit 1; }
    { awk -v r="$5" '$1 != r' "$reg"; echo "$5 $d"; } > "$reg.novo"
    mv "$reg.novo" "$reg"
    ;;
esac
`

const fakeGh = `#!/bin/bash
echo "$*" >> "$FAKE_DIR/gh.log"
while [[ $# -gt 0 && "$1" != --jq ]]; do shift; done
jq -r "$2" "$FAKE_DIR/gh.json"
`

type stepRun struct {
	dir    string
	saida  string
	falhou bool
}

func (r stepRun) arquivo(t *testing.T, nome string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(r.dir, nome))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(content)
}

func (r stepRun) digest(t *testing.T, ref string) string {
	t.Helper()
	for _, linha := range strings.Split(r.arquivo(t, "registro"), "\n") {
		if campos := strings.Fields(linha); len(campos) == 2 && campos[0] == ref {
			return campos[1]
		}
	}
	return ""
}

// stepScript devolve o bloco "run: |" do step com esse nome.
func stepScript(t *testing.T, workflow, nome string) string {
	t.Helper()
	linhas := strings.Split(workflow, "\n")
	inicio := -1
	for i, linha := range linhas {
		if strings.TrimSpace(linha) == "- name: "+nome {
			inicio = i
			break
		}
	}
	if inicio < 0 {
		t.Fatalf("step %q não encontrado", nome)
	}
	for i := inicio + 1; i < len(linhas); i++ {
		if strings.TrimSpace(linhas[i]) != "run: |" {
			if strings.HasPrefix(strings.TrimSpace(linhas[i]), "- ") {
				break
			}
			continue
		}
		indent := len(linhas[i]) - len(strings.TrimLeft(linhas[i], " ")) + 2
		var bloco []string
		for _, l := range linhas[i+1:] {
			if strings.TrimSpace(l) != "" && len(l)-len(strings.TrimLeft(l, " ")) < indent {
				break
			}
			if len(l) >= indent {
				l = l[indent:]
			}
			bloco = append(bloco, l)
		}
		return strings.Join(bloco, "\n")
	}
	t.Fatalf("step %q sem run: |", nome)
	return ""
}

// runStep roda o script como o runner sem shell: explícito (bash -e, sem pipefail), com docker e gh falsos no PATH.
func runStep(t *testing.T, script string, env map[string]string, registro, ghJSON string) stepRun {
	t.Helper()
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq não instalado")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	arquivos := map[string]string{
		filepath.Join(bin, "docker"):   fakeDocker,
		filepath.Join(bin, "gh"):       fakeGh,
		filepath.Join(dir, "registro"): registro,
		filepath.Join(dir, "gh.json"):  ghJSON,
	}
	for caminho, conteudo := range arquivos {
		if err := os.WriteFile(caminho, []byte(conteudo), 0755); err != nil {
			t.Fatal(err)
		}
	}

	cmd := exec.Command("bash", "--noprofile", "--norc", "-e", "-c", script)
	cmd.Env = []string{
		"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"FAKE_DIR=" + dir,
		"GITHUB_OUTPUT=" + filepath.Join(dir, "output"),
		"GITHUB_REPOSITORY=algarys/algarys_meu-projeto",
		"REGISTRO=ghcr.io/algarys/algarys_meu-projeto",
	}
	for chave, valor := range env {
		cmd.Env = append(cmd.Env, chave+"="+valor)
	}
	saida, err := cmd.CombinedOutput()
	if _, ok := err.(*exec.ExitError); err != nil && !ok {
		t.Fatal(err)
	}
	return stepRun{dir: dir, saida: string(saida), falhou: err != nil}
}

func renderedWorkflow(t *testing.T, nome string) string {
	t.Helper()
	return readFile(t, filepath.Join(renderDefault(t), ".github", "workflows", nome))
}

func TestDeployOrigemClassificaOPRMergeadoNaMain(t *testing.T) {
	script := stepScript(t, renderedWorkflow(t, "deploy.yml"), "PR que gerou o commit")
	const sha = "1234567890abcdef"
	pr := func(ramo string) string {
		return `[{"number": 7, "merge_commit_sha": "` + sha + `", "head": {"ref": "` + ramo + `", "sha": "abcdef0123"}}]`
	}

	casos := []struct {
		nome, ghJSON, saida string
	}{
		{"promoção usa o head da dev", pr("dev"), "tipo=promocao\nramo=dev\ntag=sha-abcdef0\n"},
		{"hotfix usa o commit da main", pr("hotfix/login"), "tipo=hotfix\nramo=hotfix/login\ntag=sha-1234567\n"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			r := runStep(t, script, map[string]string{"GITHUB_SHA": sha}, "", c.ghJSON)
			if r.falhou {
				t.Fatalf("falhou: %s", r.saida)
			}
			if got := r.arquivo(t, "output"); got != c.saida {
				t.Errorf("GITHUB_OUTPUT = %q, want %q", got, c.saida)
			}
		})
	}

	for nome, ghJSON := range map[string]string{
		"origem fora de dev e hotfix": pr("feature/x"),
		"commit sem PR":               "[]",
	} {
		t.Run(nome, func(t *testing.T) {
			r := runStep(t, script, map[string]string{"GITHUB_SHA": sha}, "", ghJSON)
			if !r.falhou || !strings.Contains(r.saida, "::error::") {
				t.Errorf("deveria falhar com ::error::, saída: %s", r.saida)
			}
		})
	}
}

const registroBase = "ghcr.io/algarys/algarys_meu-projeto-api"

func TestDeployPromoverMoveProducaoSemRebuild(t *testing.T) {
	script := stepScript(t, renderedWorkflow(t, "deploy.yml"), "producao aponta para a mesma imagem, sem rebuild")
	env := func(tipo string) map[string]string {
		return map[string]string{"TIPO": tipo, "TAG": "sha-abcdef0", "IMAGENS": "api"}
	}

	t.Run("promoção com homolog em outra tag falha antes de reetiquetar", func(t *testing.T) {
		registro := registroBase + ":sha-abcdef0 sha256:novo\n" + registroBase + ":homolog sha256:velho\n"
		r := runStep(t, script, env("promocao"), registro, "")
		if !r.falhou || !strings.Contains(r.saida, "api:homolog não é sha-abcdef0") {
			t.Errorf("deveria falhar apontando a homolog, saída: %s", r.saida)
		}
		if strings.Contains(r.arquivo(t, "docker.log"), "create") {
			t.Error("não deveria reetiquetar nenhuma imagem")
		}
	})

	t.Run("promoção correta deixa producao com o digest da tag", func(t *testing.T) {
		registro := registroBase + ":sha-abcdef0 sha256:novo\n" + registroBase + ":homolog sha256:novo\n" +
			registroBase + ":producao sha256:velho\n"
		r := runStep(t, script, env("promocao"), registro, "")
		if r.falhou {
			t.Fatalf("falhou: %s", r.saida)
		}
		if got := r.digest(t, registroBase+":producao"); got != "sha256:novo" {
			t.Errorf("producao = %q, want sha256:novo", got)
		}
	})

	t.Run("hotfix não exige homolog", func(t *testing.T) {
		registro := registroBase + ":sha-abcdef0 sha256:fix\n" + registroBase + ":homolog sha256:velho\n"
		r := runStep(t, script, env("hotfix"), registro, "")
		if r.falhou {
			t.Fatalf("falhou: %s", r.saida)
		}
		if got := r.digest(t, registroBase+":producao"); got != "sha256:fix" {
			t.Errorf("producao = %q, want sha256:fix", got)
		}
	})

	t.Run("tag inexistente falha", func(t *testing.T) {
		r := runStep(t, script, env("hotfix"), registroBase+":homolog sha256:velho\n", "")
		if !r.falhou || !strings.Contains(r.saida, "api:sha-abcdef0 não existe no GHCR") {
			t.Errorf("deveria falhar com tag inexistente, saída: %s", r.saida)
		}
	})
}

func TestDeployCobrarVoltaDevExigePRDoHotfixParaDev(t *testing.T) {
	script := stepScript(t, renderedWorkflow(t, "deploy.yml"), "PR do hotfix para a dev")
	env := map[string]string{"RAMO": "hotfix/login"}

	r := runStep(t, script, env, "", `[{"number": 9, "state": "CLOSED"}]`)
	if !r.falhou || !strings.Contains(r.saida, "::error::hotfix/login foi para a main sem PR aberto ou mergeado para a dev") {
		t.Errorf("sem PR válido deveria falhar, saída: %s", r.saida)
	}
	if log := r.arquivo(t, "gh.log"); !strings.Contains(log, "--head hotfix/login --base dev --state all") {
		t.Errorf("gh chamado com %q", log)
	}

	r = runStep(t, script, env, "", `[{"number": 9, "state": "CLOSED"}, {"number": 10, "state": "MERGED"}]`)
	if r.falhou || !strings.Contains(r.saida, "#10 (MERGED)") {
		t.Errorf("com PR mergeado deveria passar, saída: %s", r.saida)
	}
}

func TestRollbackValidaAntesDoDeploy(t *testing.T) {
	rollback := renderedWorkflow(t, "rollback.yml")

	t.Run("producao disparado da dev falha com mensagem clara", func(t *testing.T) {
		script := stepScript(t, rollback, "Branch do run bate com o ambiente")
		r := runStep(t, script, map[string]string{"AMBIENTE": "producao", "GITHUB_REF_NAME": "dev"}, "", "")
		if !r.falhou || !strings.Contains(r.saida, "::error::rollback de producao roda a partir da main, não da dev") {
			t.Errorf("saída: %s", r.saida)
		}
		r = runStep(t, script, map[string]string{"AMBIENTE": "homolog", "GITHUB_REF_NAME": "dev"}, "", "")
		if r.falhou {
			t.Errorf("homolog da dev deveria passar: %s", r.saida)
		}
	})

	t.Run("tag fora do formato sha-<7 hex>", func(t *testing.T) {
		script := stepScript(t, rollback, "Formato da tag")
		for _, tag := range []string{"homolog", "sha-abcdef", "sha-ABCDEF0", "sha-abcdef01"} {
			if r := runStep(t, script, map[string]string{"TAG": tag}, "", ""); !r.falhou {
				t.Errorf("tag %q deveria falhar", tag)
			}
		}
		if r := runStep(t, script, map[string]string{"TAG": "sha-abcdef0"}, "", ""); r.falhou {
			t.Errorf("sha-abcdef0 deveria passar: %s", r.saida)
		}
	})

	t.Run("imagem precisa existir no GHCR", func(t *testing.T) {
		script := stepScript(t, rollback, "Imagens existem no GHCR")
		env := map[string]string{"TAG": "sha-abcdef0", "IMAGENS": "api"}
		r := runStep(t, script, env, "", "")
		if !r.falhou || !strings.Contains(r.saida, "api:sha-abcdef0 não existe no GHCR") {
			t.Errorf("saída: %s", r.saida)
		}
		if r := runStep(t, script, env, registroBase+":sha-abcdef0 sha256:x\n", ""); r.falhou {
			t.Errorf("imagem existente deveria passar: %s", r.saida)
		}
	})
}

func TestRollbackMoveEtiquetaDoAmbiente(t *testing.T) {
	script := stepScript(t, renderedWorkflow(t, "rollback.yml"), "Etiqueta do ambiente aponta para a versão restaurada")
	registro := registroBase + ":sha-abcdef0 sha256:antigo\n" + registroBase + ":producao sha256:atual\n"
	r := runStep(t, script, map[string]string{"AMBIENTE": "producao", "TAG": "sha-abcdef0", "IMAGENS": "api"}, registro, "")
	if r.falhou {
		t.Fatalf("falhou: %s", r.saida)
	}
	if got := r.digest(t, registroBase+":producao"); got != "sha256:antigo" {
		t.Errorf("producao = %q, want sha256:antigo", got)
	}
}
