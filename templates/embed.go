package templates

import "embed"

//go:embed all:project
var projectFS embed.FS

func RenderProject(dest string, data Data) error {
	return render(projectFS, dest, data)
}
