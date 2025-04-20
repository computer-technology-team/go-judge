package templates

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"slices"

	internalcontext "github.com/computer-technology-team/go-judge/internal/context"
	"github.com/computer-technology-team/go-judge/internal/storage"
)

type PackageName string

const (
	Home           PackageName = "home"
	Profiles       PackageName = "profiles"
	CreateProblem  PackageName = "problems"
	Authentication PackageName = "authentication"
	Submissions    PackageName = "submissions"
)

//go:embed shared/*.gohtml shared/layouts/*.gohtml shared/partials/*.gohtml home/*.gohtml profiles/*.gohtml authentication/*.gohtml problems/*.gohtml submissions/*.gohtml
var templateFS embed.FS

// Templates holds all parsed templates
type Templates struct {
	templates map[string]*template.Template
}

type TemplateData struct {
	Data any
	User *storage.User
}

func GetSharedTemplates() (*Templates, error) {
	templates := make(map[string]*template.Template)

	shared, err := fs.Glob(templateFS, "shared/*.gohtml")
	if err != nil {
		return nil, fmt.Errorf("could not get shared templates glob: %w", err)
	}

	sharedLayouts, err := fs.Glob(templateFS, "shared/layouts/*.gohtml")
	if err != nil {
		return nil, fmt.Errorf("could not get shared layout templates glob: %w", err)
	}

	sharedPartials, err := fs.Glob(templateFS, "shared/partials/*.gohtml")
	if err != nil {
		return nil, fmt.Errorf("could not get shared partial templates glob: %w", err)
	}

	var sharedTemplates []string
	sharedTemplates = append(sharedTemplates, sharedLayouts...)
	sharedTemplates = append(sharedTemplates, sharedPartials...)

	for _, page := range shared {
		name := fileNameWithoutExt(page)

		tmpl := template.New(name)

		allTemplates := append(sharedTemplates, page)
		tmpl, err = tmpl.ParseFS(templateFS, allTemplates...)
		if err != nil {
			return nil, err
		}

		templates[name] = tmpl
	}

	return &Templates{templates: templates}, nil
}

func GetTemplates(pkg PackageName) (*Templates, error) {
	templates := make(map[string]*template.Template)

	pkgTemplates, err := fs.Glob(templateFS, fmt.Sprintf("%s/*.gohtml", string(pkg)))
	if err != nil {
		return nil, err
	}

	shared, err := fs.Glob(templateFS, "shared/*.gohtml")
	if err != nil {
		return nil, fmt.Errorf("could not get shared templates glob: %w", err)
	}

	sharedLayouts, err := fs.Glob(templateFS, "shared/layouts/*.gohtml")
	if err != nil {
		return nil, fmt.Errorf("could not get shared layout templates glob: %w", err)
	}

	sharedPartials, err := fs.Glob(templateFS, "shared/partials/*.gohtml")
	if err != nil {
		return nil, fmt.Errorf("could not get shared partial templates glob: %w", err)
	}

	var sharedTemplates []string
	sharedTemplates = append(sharedTemplates, sharedLayouts...)
	sharedTemplates = append(sharedTemplates, sharedPartials...)

	for _, page := range slices.Concat(shared, pkgTemplates) {
		name := fileNameWithoutExt(page)

		tmpl := template.New(name)

		allTemplates := append(sharedTemplates, page)
		tmpl, err = tmpl.ParseFS(templateFS, allTemplates...)
		if err != nil {
			return nil, err
		}

		templates[name] = tmpl
	}

	return &Templates{templates: templates}, nil
}

func (t *Templates) Render(ctx context.Context, name string, rw http.ResponseWriter, data any) error {
	tmpl, ok := t.templates[name]
	if !ok {
		return errors.New("could not read template")
	}

	templateData := TemplateData{Data: data}

	if user, ok := internalcontext.GetUserFromContext(ctx); ok {
		templateData.User = user
	}

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	// http response writer automatically sends 200 on write call
	err := tmpl.ExecuteTemplate(rw, name, templateData)
	if err != nil {
		return fmt.Errorf("could not execute template: %w", err)
	}

	return nil
}

func fileNameWithoutExt(page string) string {
	name := filepath.Base(page)
	name = name[0 : len(name)-len(filepath.Ext(name))]
	return name
}
