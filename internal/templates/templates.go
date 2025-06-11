// Package templates provides workspace template files for AI agent context.
package templates

import (
	_ "embed"
	"os"
	"path/filepath"
	"text/template"
)

// Embedded template files
var (
	//go:embed background.md
	BackgroundTemplate string

	//go:embed instructions.md
	InstructionsTemplate string

	//go:embed plan.md
	PlanTemplate string

	//go:embed working-log.md
	WorkingLogTemplate string

	//go:embed results-summary.md
	ResultsSummaryTemplate string
)

// TemplateData represents data for template rendering
type TemplateData struct {
	ProjectName string
	WorkspaceID string
	Branch      string
	AgentID     string
	Timestamp   string
}

// WriteContextFiles writes the working context files to a workspace
func WriteContextFiles(workspacePath string, data TemplateData) error {
	contextDir := filepath.Join(workspacePath, ".amux", "context")
	if err := os.MkdirAll(contextDir, 0o755); err != nil {
		return err
	}

	files := map[string]string{
		"background.md":      BackgroundTemplate,
		"plan.md":            PlanTemplate,
		"working-log.md":     WorkingLogTemplate,
		"results-summary.md": ResultsSummaryTemplate,
	}

	for filename, content := range files {
		if err := writeTemplate(filepath.Join(contextDir, filename), content, data); err != nil {
			return err
		}
	}

	return nil
}

// WriteInstructions writes the instructions.md file to a workspace
func WriteInstructions(workspacePath string, data TemplateData) error {
	amuxDir := filepath.Join(workspacePath, ".amux")
	if err := os.MkdirAll(amuxDir, 0o755); err != nil {
		return err
	}

	return writeTemplate(filepath.Join(amuxDir, "instructions.md"), InstructionsTemplate, data)
}

func writeTemplate(path, templateContent string, data TemplateData) error {
	tmpl, err := template.New("template").Parse(templateContent)
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	err = tmpl.Execute(file, data)
	return err
}
