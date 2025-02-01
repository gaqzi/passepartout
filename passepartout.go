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
	all := map[string]tmpl{}
	var templts []string
	templates := map[string]*template.Template{}
	layouts := map[string]string{}

	err := fs.WalkDir(fsys, ".", func(p string, entry fs.DirEntry, err error) error {
		if entry.IsDir() {
			all[p+"/"] = tmpl{}
			return nil // skip since WalkDir will recurse for us
		}

		if strings.HasPrefix(entry.Name(), "_") {
			filePath := path.Dir(p) + "/"
			x, ok := all[filePath]
			if !ok {
				return fmt.Errorf("failed to find template %q", filePath) // XXX: add test case for this (can I?)
			}
			x.partials = append(x.partials, p)
			all[filePath] = x
			return nil
		} else if isLayout := strings.HasPrefix(p, "layouts/") || strings.Contains(p, "/layouts/"); isLayout {
			content, err := fs.ReadFile(fsys, p)
			if err != nil {
				return fmt.Errorf("failed to read layout %q: %w", p, err)
			}

			layouts[p] = string(content)
			return nil
		}

		templts = append(templts, p)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, t := range templts {
		baseWithPartials, err := baseTemplate.Clone()
		if err != nil {
			return nil, fmt.Errorf("failed to clone base template: %w", err)
		}

		dir := path.Dir(t) + "/"
		if sameDir, ok := all[dir]; ok {
			for _, partial := range sameDir.partials {
				content, err := fs.ReadFile(fsys, partial)
				if err != nil {
					return nil, fmt.Errorf("failed to read partial %q: %w", partial, err)
				}
				_, err = baseWithPartials.New(path.Base(partial)).Parse(string(content))
				if err != nil {
					return nil, fmt.Errorf("failed to parse partial %q: %w", partial, err)
				}
			}

		}

		ext := path.Ext(t)
		tmplDir := strings.TrimSuffix(t, ext) + "/"
		if partialDir, ok := all[tmplDir]; ok {
			for _, partial := range partialDir.partials {
				content, err := fs.ReadFile(fsys, partial)
				if err != nil {
					return nil, fmt.Errorf("failed to read partial %q: %w", partial, err)
				}
				_, err = baseWithPartials.New(path.Base(partial)).Parse(string(content))
				if err != nil {
					return nil, fmt.Errorf("failed to parse partial %q: %w", partial, err)
				}
			}
		}

		content, err := fs.ReadFile(fsys, t)
		if err != nil {
			return nil, fmt.Errorf("failed to read template %q: %w", t, err)
		}

		// add the page template without a layout
		base, err := baseWithPartials.Clone()
		if err != nil {
			return nil, fmt.Errorf("failed to clone template %q: %w", t, err)
		}
		if _, err := base.New(t).Parse(string(content)); err != nil {
			return nil, fmt.Errorf("failed to parse template %q: %w", t, err)
		}
		templates[t] = base

		// add the page wrapped in a block content with a prefix path for each layout
		contentInContent := `{{ define "content" }}` + string(content) + `{{ end }}`
		for l, lContent := range layouts {
			base, err := baseWithPartials.Clone()
			if err != nil {
				return nil, fmt.Errorf("failed to clone template %q: %w", t, err)
			}

			if _, err := base.New(l).Parse(lContent); err != nil {
				return nil, fmt.Errorf("failed to parse layout %q: %w", l, err)
			}

			layoutPagePath := path.Join(l, t)
			if _, err := base.New(layoutPagePath).Parse(contentInContent); err != nil {
				return nil, fmt.Errorf("failed to parse template when wrapped in define %q: %w", t, err)
			}

			templates[layoutPagePath] = base
		}
	}

	pp := Passepartout{Templates: templates}
	for _, option := range options {
		option(&pp)
	}

	return &pp, nil
}
