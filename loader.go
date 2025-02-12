package passepartout

import (
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"path"
	"strings"
)

type FileWithContent struct {
	Name    string
	Content string
}

func Template(base *template.Template, files ...FileWithContent) (*template.Template, error) {
	if base == nil {
		base = template.New("")
	}

	for _, file := range files {
		if _, err := base.New(file.Name).Parse(file.Content); err != nil {
			return nil, fmt.Errorf("failed to parse %q into the template: %w", file.Name, err)
		}
	}

	return base, nil
}

// PartialLoader loads all the partials for a page/component and returns a slice of FileWithContent.
type PartialLoader func(page string) ([]FileWithContent, error)

// TemplateLoader loads a page/component into a template and knows how to load layouts together for a page/component.
type TemplateLoader interface {
	Page(name string) ([]FileWithContent, error)
	PageInLayout(name string, layout string) ([]FileWithContent, error)
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

type Loader struct {
	PartialsFor    PartialLoader
	TemplateLoader TemplateLoader
	CreateTemplate Templater
}

func (l *Loader) Page(name string) (*template.Template, error) {
	files, err := flatMap(name, l.PartialsFor, l.TemplateLoader.Page)
	if err != nil {
		return nil, fmt.Errorf("failed to collect all files for page %q: %w", name, err)
	}

	tmplt, err := l.CreateTemplate(nil, files)
	if err != nil {
		return nil, fmt.Errorf("failed to create template for page %q: %w", name, err)
	}

	return tmplt, nil
}

type PartialsInFolderOnly struct {
	FS fs.ReadDirFS
}

func (p *PartialsInFolderOnly) Load(page string) ([]FileWithContent, error) {
	ext := path.Ext(page)
	dirName := strings.TrimSuffix(page, ext)

	var files []FileWithContent
	err := fs.WalkDir(p.FS, dirName, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}

		if entry.IsDir() {
			return nil
		}

		content, err := fs.ReadFile(p.FS, filePath)
		if err != nil {
			return err
		}

		files = append(files, FileWithContent{Name: filePath, Content: string(content)})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

type TemplateByNameLoader struct {
	FS fs.ReadFileFS
}

func (t *TemplateByNameLoader) Page(name string) ([]FileWithContent, error) {
	content, err := t.FS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	return []FileWithContent{{Name: name, Content: string(content)}}, nil
}

func (t *TemplateByNameLoader) PageInLayout(name, layout string) ([]FileWithContent, error) {
	pages, err := t.Page(name)
	if err != nil {
		return nil, err
	}

	layoutContent, err := t.FS.ReadFile(layout)
	if err != nil {
		return nil, fmt.Errorf("failed to read layout template: %w", err)
	}

	pages = append(pages, FileWithContent{Name: layout, Content: string(layoutContent)})
	return pages, nil
}
