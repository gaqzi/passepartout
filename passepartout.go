package passepartout

import (
	"errors"
	"io"
	"io/fs"
)

type readDirReadFileFS interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

// FSWithoutPrefix will take a passed in filesystem and strip away "prefix" when using the filesystem.
// It uses [fs.Sub] under the hood, and it's a wrapper to ensure the returned filesystem can be used by passepartout.
// The usecase is that you store all your templates in `templates/` and don't want to actually use your templates as
// `templates/page/index.tmpl` and instead just say `page/index.html`.
func FSWithoutPrefix(fs_ readDirReadFileFS, prefix string) (readDirReadFileFS, error) {
	sub, err := fs.Sub(fs_, prefix)
	if err != nil {
		return nil, err
	}

	fs_, ok := sub.(readDirReadFileFS)
	if !ok {
		return nil, errors.New("[fs.Sub] returned a filesystem that doesn't implement readDirReadFileFS, this is probably a bug in passepartout")
	}

	return fs_, nil
}

type passepartout struct {
	loader *Loader
}

// Load initializes a template manager to load and render templates within a passed in filesystem.
// Passepartout manages Go templates and their relationships to make sure they "pop" and are simple to manage.
// It does this by relying on a hierarchy in a folder that is:
//
//	templates/                               # base folder
//	templates/layouts                        # The layouts that "pages" or "components" that are rendered within
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
//	str, err := passepartout.Render("index/main.tmpl", map[string]any{"Items": []string{"Hello", "World"}})  // renders the index/main.tmpl using the index/_main/_item.tmpl partial and returns the result as a string
func Load(fs_ readDirReadFileFS) (*passepartout, error) {
	partials := &PartialsInFolderOnly{FS: fs_}
	l := Loader{
		PartialsFor:    partials.Load,
		TemplateLoader: &TemplateByNameLoader{FS: fs_},
		CreateTemplate: CreateTemplate,
	}

	return &passepartout{loader: &l}, nil
}

func (p *passepartout) Render(out io.Writer, name string, data any) error {
	t, err := p.loader.Page(name)
	if err != nil {
		return err
	}

	return t.ExecuteTemplate(out, name, data)
}

func (p *passepartout) RenderInLayout(out io.Writer, layout string, name string, data any) error {
	t, err := p.loader.InLayout(name, layout)
	if err != nil {
		return err
	}

	return t.ExecuteTemplate(out, layout, data)
}
