package ppdefaults

import (
	"fmt"
	"html/template"
	"io/fs"
)

type FileWithContent struct {
	Name    string
	Content string
}

// PartialLoader loads all the partials for a template and returns a slice of FileWithContent.
type PartialLoader func(page string) ([]FileWithContent, error)

// TemplateLoader loads a template and knows how to templates for use in a layout.
type TemplateLoader interface {
	Standalone(name string) ([]FileWithContent, error)
	InLayout(name string, layout string) ([]FileWithContent, error)
}

// Templater either creates a new template with name and content or adds that template to an existing collection of templates.
type Templater func(base *template.Template, files []FileWithContent) (*template.Template, error)

// FS specifies which filesystems we need to be able to work.
type FS interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

func flatMap(name string, fns ...func(string) ([]FileWithContent, error)) ([]FileWithContent, error) {
	var files []FileWithContent

	for _, fn := range fns {
		result, err := fn(name)
		if err != nil {
			return nil, err
		}
		files = append(files, result...)
	}

	return files, nil
}

// WithDefaults sets the default Partial and Template loader together with the template creator using the passed in FS.
// Uses:
//   - [PartialsInFolderOnly] for PartialsFor
//   - [TemplateByNameLoader] for TemplateLoader
//   - [CreateTemplate] for CreateTemplate
func (b *LoaderBuilder) WithDefaults(fsys FS) *LoaderBuilder {
	partials := PartialsInFolderOnly{FS: fsys}
	b.build.PartialsFor = partials.Load

	b.build.TemplateLoader = &TemplateByNameLoader{FS: fsys}
	b.build.CreateTemplate = CreateTemplate

	return b
}

type Loader struct {
	// TemplateConfig is used as a base when creating new templates from a collection of files.
	// See [template.Template.Funcs] and [template.Template.Option] for what often is configured.
	TemplateConfig *template.Template
	PartialsFor    PartialLoader
	TemplateLoader TemplateLoader
	CreateTemplate Templater
}

func (l *Loader) Standalone(name string) (*template.Template, error) {
	files, err := flatMap(name, l.PartialsFor, l.TemplateLoader.Standalone)
	if err != nil {
		return nil, fmt.Errorf("failed to collect all files for %q: %w", name, err)
	}

	tmplt, err := l.CreateTemplate(l.TemplateConfig, files)
	if err != nil {
		return nil, fmt.Errorf("failed to create template for %q: %w", name, err)
	}

	return tmplt, nil
}

func (l *Loader) InLayout(page string, layout string) (*template.Template, error) {
	var files []FileWithContent
	partials, err := l.PartialsFor(page)
	if err != nil {
		return nil, fmt.Errorf("failed to collect partials for %q: %w", page, err)
	}
	files = append(files, partials...)

	pageFiles, err := l.TemplateLoader.InLayout(page, layout)
	if err != nil {
		return nil, fmt.Errorf("failed to collect all for %q in layout %q: %w", page, layout, err)
	}
	files = append(files, pageFiles...)

	tmplt, err := l.CreateTemplate(l.TemplateConfig, files)
	if err != nil {
		return nil, fmt.Errorf("failed to create template for %q in layout %q: %w", page, layout, err)
	}

	return tmplt, nil
}

type TemplateByNameLoader struct {
	FS fs.ReadFileFS
}

func (t *TemplateByNameLoader) Standalone(name string) ([]FileWithContent, error) {
	content, err := t.FS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	return []FileWithContent{{Name: name, Content: string(content)}}, nil
}

func (t *TemplateByNameLoader) InLayout(name, layout string) ([]FileWithContent, error) {
	pages, err := t.Standalone(name)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(pages); i++ {
		pages[i].Content = `{{ define "content" }}` + pages[i].Content + `{{ end }}`
	}

	layoutContent, err := t.FS.ReadFile(layout)
	if err != nil {
		return nil, fmt.Errorf("failed to read layout template: %w", err)
	}

	// Intentionally prepend the layout so any declared definitions from it will be overridden by other templates,
	// for example `{{ define "HEADER" }}` or similar blocks. If not, the default provided by the template will be the
	// last one defined, and therefore used.
	pages = append([]FileWithContent{{Name: layout, Content: string(layoutContent)}}, pages...)
	return pages, nil
}

func CreateTemplate(base *template.Template, files []FileWithContent) (*template.Template, error) {
	var tmplt *template.Template
	var err error
	if base != nil {
		tmplt, err = base.Clone()
		if err != nil {
			return nil, fmt.Errorf("failed to copy base template: %w", err)
		}
	} else {
		tmplt = template.New("")
	}

	for _, file := range files {
		if _, err := tmplt.New(file.Name).Parse(file.Content); err != nil {
			return nil, fmt.Errorf("failed to parse template: %w", err)
		}
	}

	return tmplt, nil
}
