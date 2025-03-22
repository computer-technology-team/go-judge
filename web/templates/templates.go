package templates

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path/filepath"
)

type PackageName string

const Home PackageName = "home"

//go:embed shared/layouts/*.gohtml shared/partials/*.gohtml home/*.gohtml
var templateFS embed.FS

// Templates holds all parsed templates
type Templates struct {
	templates map[string]*template.Template
}

type TemplateData struct {
	Data any
}

// New creates a new Templates instance with all templates parsed
func GetTemplates(pkg PackageName) (*Templates, error) {
	templates := make(map[string]*template.Template)

	// Get all template files
	pkgTemplates, err := fs.Glob(templateFS, fmt.Sprintf("%s/*.gohtml", string(pkg)))
	if err != nil {
		return nil, err
	}

	// Get all shared templates for reuse
	sharedLayouts, err := fs.Glob(templateFS, "shared/layouts/*.gohtml")
	if err != nil {
		return nil, err
	}

	sharedPartials, err := fs.Glob(templateFS, "shared/partials/*.gohtml")
	if err != nil {
		return nil, err
	}

	// Combine all shared templates
	var sharedTemplates []string
	sharedTemplates = append(sharedTemplates, sharedLayouts...)
	sharedTemplates = append(sharedTemplates, sharedPartials...)

	// Parse each page template
	for _, page := range pkgTemplates {
		name := filepath.Base(page)
		name = name[0 : len(name)-len(filepath.Ext(name))]

		// Create a template with the base layout, partials, and all other templates
		tmpl := template.New(name)

		// Parse all shared templates and the page template
		allTemplates := append(sharedTemplates, page)
		tmpl, err = tmpl.ParseFS(templateFS, allTemplates...)
		if err != nil {
			return nil, err
		}

		templates[name] = tmpl
	}

	return &Templates{templates: templates}, nil
}

func (t *Templates) Render(ctx context.Context, name string, wr io.Writer, data any) error {
	tmpl, ok := t.templates[name]
	if !ok {
		return errors.New("could not read template")
	}

	return tmpl.Execute(wr, TemplateData{Data: data})
}
