package templates

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ler %s: %v", path, err)
	}
	return string(content)
}

func TestRenderSubstitutesPlaceholders(t *testing.T) {
	fsys := fstest.MapFS{
		"project/.github/info.txt.tmpl": {Data: []byte("<% .Projeto %>|<% .Repositorio %>|<% .Registro %>|${{ github.sha }}\n")},
	}
	dest := t.TempDir()

	if err := render(fsys, dest, NewData("meu-projeto", false)); err != nil {
		t.Fatalf("render() error = %v", err)
	}

	got := readFile(t, filepath.Join(dest, ".github", "info.txt"))
	want := "meu-projeto|algarys_meu-projeto|ghcr.io/algarys/algarys_meu-projeto|${{ github.sha }}\n"
	if got != want {
		t.Errorf("conteúdo = %q, want %q", got, want)
	}
}

func TestRenderSkipsFrontendWithoutFrontend(t *testing.T) {
	fsys := fstest.MapFS{
		"project/README.md.tmpl":             {Data: []byte("readme")},
		"project/frontend/package.json.tmpl": {Data: []byte("{}")},
	}

	for _, temFrontend := range []bool{false, true} {
		dest := t.TempDir()
		if err := render(fsys, dest, NewData("p", temFrontend)); err != nil {
			t.Fatalf("render() error = %v", err)
		}
		if _, err := os.Stat(filepath.Join(dest, "README.md")); err != nil {
			t.Errorf("README.md deveria existir sempre: %v", err)
		}
		_, err := os.Stat(filepath.Join(dest, "frontend", "package.json"))
		if gotFrontend := err == nil; gotFrontend != temFrontend {
			t.Errorf("TemFrontend=%v: frontend gerado = %v", temFrontend, gotFrontend)
		}
	}
}
