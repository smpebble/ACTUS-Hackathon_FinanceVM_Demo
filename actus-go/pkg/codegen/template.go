package codegen

import (
	"fmt"
	"io"
	"path/filepath"
	"text/template"
)

// DefaultTemplateRenderer implements TemplateRenderer using Go's text/template
type DefaultTemplateRenderer struct {
	templates map[string]*template.Template
	funcMap   template.FuncMap
}

// NewTemplateRenderer creates a new template renderer
func NewTemplateRenderer() TemplateRenderer {
	return &DefaultTemplateRenderer{
		templates: make(map[string]*template.Template),
		funcMap:   createDefaultTemplateFuncMap(),
	}
}

// createDefaultTemplateFuncMap creates helper functions available in templates
func createDefaultTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"toLower": func(s string) string {
			return s
		},
		"toUpper": func(s string) string {
			return s
		},
		"capitalize": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return string(s[0]-32) + s[1:]
		},
		"formatDecimal": func(d interface{}) string {
			return fmt.Sprintf("%v", d)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
	}
}

// Render executes a template with the given data
func (r *DefaultTemplateRenderer) Render(templateName string, data interface{}) (string, error) {
	tmpl, exists := r.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	var buf io.Writer
	err := tmpl.Execute(buf, data)
	if err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}

	return buf.(fmt.Stringer).String(), nil
}

// RenderToWriter executes a template and writes to an io.Writer
func (r *DefaultTemplateRenderer) RenderToWriter(w io.Writer, templateName string, data interface{}) error {
	tmpl, exists := r.templates[templateName]
	if !exists {
		return fmt.Errorf("template not found: %s", templateName)
	}

	err := tmpl.Execute(w, data)
	if err != nil {
		return fmt.Errorf("template execution failed: %w", err)
	}

	return nil
}

// LoadTemplates loads templates from a directory
func (r *DefaultTemplateRenderer) LoadTemplates(dir string) error {
	pattern := filepath.Join(dir, "*.tmpl")

	tmpl, err := template.New("").Funcs(r.funcMap).ParseGlob(pattern)
	if err != nil {
		return fmt.Errorf("failed to load templates from %s: %w", dir, err)
	}

	// Store individual templates
	for _, t := range tmpl.Templates() {
		r.templates[t.Name()] = t
	}

	return nil
}

// AddTemplate adds a single template by name and content
func (r *DefaultTemplateRenderer) AddTemplate(name, content string) error {
	tmpl, err := template.New(name).Funcs(r.funcMap).Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	r.templates[name] = tmpl
	return nil
}
