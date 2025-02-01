package passepartout

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path"
	"strings"
)

type Option func(p *Passepartout)

// Passepartout manages Go templates and their relationships to make sure they "pop" and are simple to manage.
//
// It does this by relying on a hierarchy in a folder that is:
//
//	templates/                               # base folder
//	templates/layouts                        # The layouts that "pages" or "components" that are rendered within
//	templates/partials                       # Global partials that are available for all pages
//	templates/<domain>/                      # A domain where we have one or more pages, e.g. "reviews"
//	templates/<domain>/<name>.<ext>          # A page within the domain, e.g. "reviews/index.tmpl"
//	templates/<domain>/<name>/_<name>.<ext>  # A partial or a portion of a page, something that's split up
//	                                         # for reuse or organization. Partials might even exist in folders
//	                                         # if they are big.
//
// When a page template has a folder with the same name as itself (without the extension) then all partials in that
// folder is loaded alongside the template.
//
// Each template is named after its path except for the template folder's prefix, which is removed.
// Ex: "templates/reviews/show/_contributing-causes.tmpl" is named "reviews/show/_contributing-causes.tmpl"
//
// Passepartout works with embeddings and takes a filesystem to use when searching and loading templates.
//
// Given the folder structure:
//
//	templates/index/main.tmpl
//	templates/index/_main/_item.tmpl
//
// Usage:
//
//	passepartout := passepartout.Load(os.DirFS("templates/")) // the path to the base folder, removes the first part so all templates are referenced out of this folder
//	templates := passepartout.Templates() // returns a list of all known templates and a mapping of which partials it has access to. Ex: {"index/main.tmpl": []string{"index/_main/_item.tmpl"}}
//	str, err := passepartout.Render("index/main.tmpl", map[string]any{"Items": []string{"Hello", "World"}})  // renders the index/main.tmpl using the index/_main/_item.tmpl partial and returns the result as a string
type Passepartout struct {
	Templates map[string]*template.Template
}

func (p *Passepartout) Render(out io.Writer, name string, data any) error {
	t, ok := p.Templates[name]
	if !ok {
		return fmt.Errorf("template not found: %q", name)
	}

	return t.ExecuteTemplate(out, name, data)
}

func (p *Passepartout) RenderInLayout(out io.Writer, layout string, name string, data any) error {
	t, ok := p.Templates[path.Join(layout, name)]
	if !ok {
		return fmt.Errorf("template not found: %q", name)
	}

	return t.ExecuteTemplate(out, layout, data)
}

type tmpl struct {
	name     string
	partials []string
}

// Load initializes and loads templates from the provided filesystem.
func Load(fsys fs.ReadDirFS, options ...Option) (*Passepartout, error) {
	baseTemplate := template.New("")
	load := &loader{
		fsys:     fsys,
		allFiles: map[string]*tmpl{},
		layouts:  []string{},
		pages:    []string{},
	}
	templates := map[string]*template.Template{}

	// find all the pages, partials, and layouts
	err := load.filesAndCategorize()
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// create all pages and attach their needed partials. creates a base page without layout, and then each layout we find will get a version of each page.
	for _, t := range load.pages {
		baseWithPartials, err := baseTemplate.Clone()
		if err != nil {
			return nil, fmt.Errorf("failed to clone base template: %w", err)
		}

		if err := load.sameDirPartialsIntoBase(t, baseWithPartials); err != nil {
			return nil, err
		}

		if err := load.pageFolderPartialsIntoBase(t, baseWithPartials); err != nil {
			return nil, err
		}

		// load the file itself
		pageContent, err := load.ReadFile(t)
		if err != nil {
			return nil, fmt.Errorf("failed to read template %q: %w", t, err)
		}

		base, err := baseWithPartials.Clone()
		if err != nil {
			return nil, fmt.Errorf("failed to clone template %q: %w", t, err)
		}

		// add the page without a layout
		if _, err := base.New(t).Parse(string(pageContent)); err != nil {
			return nil, fmt.Errorf("failed to parse template %q: %w", t, err)
		}
		templates[t] = base

		// add the page wrapped in a block content with a prefix path for each layout
		if err := load.pageInLayouts(pageContent, t, base, templates); err != nil {
			return nil, err
		}
	}

	pp := Passepartout{Templates: templates}
	for _, option := range options {
		option(&pp)
	}

	return &pp, nil
}

type loader struct {
	fsys      fs.ReadDirFS
	allFiles  map[string]*tmpl              // stores all the known pieces from the filesystem, the template, the folders, and all the partials
	layouts   []string                      // the known layouts with their content
	pages     []string                      // all the standalone pages that uses partials and layouts
	templates map[string]*template.Template // the templates we're making available after loading
}

func (l *loader) filesAndCategorize() error {
	return fs.WalkDir(l.fsys, ".", func(p string, entry fs.DirEntry, err error) error {
		if entry.IsDir() {
			l.allFiles[p+"/"] = &tmpl{}
		} else if isPartial := strings.HasPrefix(entry.Name(), "_"); isPartial {
			filePath := path.Dir(p) + "/"
			dir, ok := l.allFiles[filePath]
			if !ok {
				dir = &tmpl{}
				l.allFiles[filePath] = dir
			}

			dir.partials = append(dir.partials, p)
		} else if isLayout := strings.HasPrefix(p, "layouts/") || strings.Contains(p, "/layouts/"); isLayout {
			l.layouts = append(l.layouts, p)
		} else {
			l.pages = append(l.pages, p)
		}

		return nil
	})
}

func (l *loader) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(l.fsys, name)
}

func (l *loader) sameDirPartialsIntoBase(name string, base *template.Template) error {
	dir := path.Dir(name) + "/"
	if sameDir, ok := l.allFiles[dir]; ok {
		for _, partial := range sameDir.partials {
			content, err := l.ReadFile(partial)
			if err != nil {
				return fmt.Errorf("failed to read partial %q: %w", partial, err)
			}

			if _, err = base.New(path.Base(partial)).Parse(string(content)); err != nil {
				return fmt.Errorf("failed to parse partial %q: %w", partial, err)
			}
		}
	}

	return nil
}

func (l *loader) pageFolderPartialsIntoBase(name string, base *template.Template) error {
	ext := path.Ext(name)
	tmplDir := strings.TrimSuffix(name, ext) + "/"
	if partialDir, ok := l.allFiles[tmplDir]; ok {
		for _, partial := range partialDir.partials {
			content, err := l.ReadFile(partial)
			if err != nil {
				return fmt.Errorf("failed to read partial %q: %w", partial, err)
			}

			if _, err = base.New(path.Base(partial)).Parse(string(content)); err != nil {
				return fmt.Errorf("failed to parse partial %q: %w", partial, err)
			}
		}
	}

	return nil
}

func (l *loader) pageInLayouts(pageContent []byte, name string, base *template.Template, templates map[string]*template.Template) error {
	contentInContent := `{{ define "content" }}` + string(pageContent) + `{{ end }}`
	for _, lName := range l.layouts {
		page, err := base.Clone()
		if err != nil {
			return fmt.Errorf("failed to clone template for layout %q: %w", name, err)
		}

		lContent, err := l.ReadFile(lName)
		if err != nil {
			return fmt.Errorf("failed to read layout %q: %w", lName, err)
		}

		if _, err := page.New(lName).Parse(string(lContent)); err != nil {
			return fmt.Errorf("failed to parse layout %q: %w", lName, err)
		}

		layoutPagePath := path.Join(lName, name)
		if _, err := page.New(layoutPagePath).Parse(contentInContent); err != nil {
			return fmt.Errorf("failed to parse template when wrapped in define %q: %w", name, err)
		}

		templates[layoutPagePath] = page
	}

	return nil
}
