package templates

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	templateRoot   = "project"
	templateSuffix = ".tmpl"
)

// Delimitadores fora do padrão para não colidir com ${{ }} do GitHub Actions
// e [[ ]] do bash.
const leftDelim, rightDelim = "<%", "%>"

type Data struct {
	Projeto     string
	Repositorio string
	Registro    string
	TemFrontend bool
}

// conditionalDirs lista diretórios que só são gerados quando a condição vale.
var conditionalDirs = map[string]func(Data) bool{
	"frontend": func(d Data) bool { return d.TemFrontend },
}

func NewData(projeto string, temFrontend bool) Data {
	repositorio := "algarys_" + projeto
	return Data{
		Projeto:     projeto,
		Repositorio: repositorio,
		Registro:    "ghcr.io/algarys/" + repositorio,
		TemFrontend: temFrontend,
	}
}

func render(fsys fs.FS, dest string, data Data) error {
	return fs.WalkDir(fsys, templateRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative := strings.TrimPrefix(path, templateRoot+"/")
		if entry.IsDir() {
			include, conditional := conditionalDirs[relative]
			if conditional && !include(data) {
				return fs.SkipDir
			}
			return nil
		}
		relative = strings.TrimSuffix(relative, templateSuffix)
		return renderFile(fsys, path, filepath.Join(dest, filepath.FromSlash(relative)), data)
	})
}

func renderFile(fsys fs.FS, source, target string, data Data) error {
	raw, err := fs.ReadFile(fsys, source)
	if err != nil {
		return fmt.Errorf("ler template %s: %w", source, err)
	}
	tmpl, err := template.New(source).Delims(leftDelim, rightDelim).Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return fmt.Errorf("template %s: %w", source, err)
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return fmt.Errorf("renderizar %s: %w", source, err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("criar diretório de %s: %w", target, err)
	}
	if err := os.WriteFile(target, out.Bytes(), 0644); err != nil {
		return fmt.Errorf("escrever %s: %w", target, err)
	}
	return nil
}
