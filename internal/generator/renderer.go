package generator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// TemplateRenderer handles rendering of external template files
type TemplateRenderer struct{}

// NewTemplateRenderer creates a new template renderer
func NewTemplateRenderer() *TemplateRenderer {
	return &TemplateRenderer{}
}

// Render renders a template file with the provided data and writes to outputPath
func (r *TemplateRenderer) Render(ctx context.Context, templatePath string, data interface{}, outputPath string) error {
	// Read template file
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file '%s': %w", templatePath, err)
	}

	// Parse template
	tmpl, err := template.New(filepath.Base(templatePath)).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template '%s': %w", templatePath, err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory '%s': %w", outputDir, err)
	}

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file '%s': %w", outputPath, err)
	}
	defer outFile.Close()

	// Execute template
	if err := tmpl.Execute(outFile, data); err != nil {
		return fmt.Errorf("failed to execute template '%s': %w", templatePath, err)
	}

	return nil
}

// RenderWithFuncs renders a template with custom functions
func (r *TemplateRenderer) RenderWithFuncs(ctx context.Context, templatePath string, data interface{}, outputPath string, funcMap template.FuncMap) error {
	// Read template file
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file '%s': %w", templatePath, err)
	}

	// Parse template with custom functions
	tmpl, err := template.New(filepath.Base(templatePath)).Funcs(funcMap).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template '%s': %w", templatePath, err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory '%s': %w", outputDir, err)
	}

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file '%s': %w", outputPath, err)
	}
	defer outFile.Close()

	// Execute template
	if err := tmpl.Execute(outFile, data); err != nil {
		return fmt.Errorf("failed to execute template '%s': %w", templatePath, err)
	}

	return nil
}

// RenderString renders a template string (not from file) with the provided data
func (r *TemplateRenderer) RenderString(ctx context.Context, templateName string, templateStr string, data interface{}, outputPath string) error {
	// Parse template string
	tmpl, err := template.New(templateName).Parse(templateStr)
	if err != nil {
		return fmt.Errorf("failed to parse template string: %w", err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory '%s': %w", outputDir, err)
	}

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file '%s': %w", outputPath, err)
	}
	defer outFile.Close()

	// Execute template
	if err := tmpl.Execute(outFile, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
