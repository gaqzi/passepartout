package passepartout_test

import (
	"bytes"
	"errors"
	"html/template"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"passepartout"
)

type templateLoaderMock struct {
	mock.Mock
}

func (t *templateLoaderMock) Page(name string) ([]passepartout.FileWithContent, error) {
	args := t.Called(name)
	return args.Get(0).([]passepartout.FileWithContent), args.Error(1)
}

func (t *templateLoaderMock) PageInLayout(name string, layout string) ([]passepartout.FileWithContent, error) {
	args := t.Called(name, layout)
	return args.Get(0).([]passepartout.FileWithContent), args.Error(1)
}

func page(name, content string, t *templateLoaderMock) {
	t.
		On("Page", name).
		Return([]passepartout.FileWithContent{{Name: name, Content: content}}, nil)
}

func pageInLayout(page, layout string, t *templateLoaderMock, templates ...passepartout.FileWithContent) {

	t.
		On("PageInLayout", page, layout).
		Return(templates, nil)
}

func partialsFor(t *testing.T, name string, files ...passepartout.FileWithContent) func(string) ([]passepartout.FileWithContent, error) {
	t.Helper()
	return func(page string) ([]passepartout.FileWithContent, error) {
		t.Helper()
		require.Equal(t, name, page, "expected to have called PartialsFor with the name passed to Page")

		return files, nil
	}
}

func createTemplate(t *testing.T, base *template.Template, files []passepartout.FileWithContent, tmpl *template.Template) func(base *template.Template, files []passepartout.FileWithContent) (*template.Template, error) {
	t.Helper()
	return func(inBase *template.Template, inFiles []passepartout.FileWithContent) (*template.Template, error) {
		require.Equal(t, base, inBase, "expected to have called CreateTemplate with the base template")
		require.Equal(t, files, inFiles, "expected to have called CreateTemplate with the files from PartialsFor and TemplateLoader.Page")

		return tmpl, nil
	}
}

func errContains(s string) func(t *testing.T, actual *template.Template, err error) {
	return func(t *testing.T, actual *template.Template, err error) {
		t.Helper()
		require.ErrorContains(t, err, s, "expected to have wrapped the error")
		require.Nil(t, actual, "expected to have returned nil on error")
	}
}

func noTemplate(base *template.Template, files []passepartout.FileWithContent) (*template.Template, error) {
	return nil, nil
}

func TestLoader_Page(t *testing.T) {
	for _, tc := range []struct {
		name           string
		pageName       string
		partialsFor    func(string) ([]passepartout.FileWithContent, error)
		loadPage       func(tmplMock *templateLoaderMock)
		createTemplate func(base *template.Template, files []passepartout.FileWithContent) (*template.Template, error)
		expect         func(t *testing.T, actual *template.Template, err error)
	}{
		{
			name:        "with no errors and referencing a partial a useful template is returned",
			pageName:    "test.tmpl",
			partialsFor: partialsFor(t, "test.tmpl", passepartout.FileWithContent{Name: "_example.tmpl", Content: "- an example partial!"}),
			loadPage:    func(tmplMock *templateLoaderMock) { page("test.tmpl", "Hello, world!", tmplMock) },
			createTemplate: createTemplate(
				t,
				nil,
				[]passepartout.FileWithContent{{Name: "_example.tmpl", Content: "- an example partial!"}, {Name: "test.tmpl", Content: "Hello, world!"}},
				template.Must(template.New("test.tmpl").Parse("Greetings world!")),
			),
			expect: func(t *testing.T, actual *template.Template, err error) {
				require.NoError(t, err)
				buf := new(bytes.Buffer)
				require.NoError(t, actual.Execute(buf, nil), "expected to be able to execute the returned template")
				require.Equal(t, "Greetings world!", buf.String())
			},
		},
		{
			name:     "when loading partials fails, the error is returned",
			pageName: "test.tmpl",
			partialsFor: func(page string) ([]passepartout.FileWithContent, error) {
				return nil, errors.New("uh-oh partial error")
			},
			loadPage:       func(tmplMock *templateLoaderMock) {},
			createTemplate: noTemplate,
			expect:         errContains(`failed to collect all files for page "test.tmpl": uh-oh partial error`),
		},
		{
			name:        "when loading the template fails, the error is returned",
			pageName:    "test.tmpl",
			partialsFor: partialsFor(t, "test.tmpl"),
			loadPage: func(tmplMock *templateLoaderMock) {
				tmplMock.On("Page", "test.tmpl").
					Return([]passepartout.FileWithContent(nil), errors.New("uh-oh template error"))
			},
			createTemplate: noTemplate,
			expect:         errContains(`failed to collect all files for page "test.tmpl": uh-oh template error`),
		},
		{
			name:        "when creating the template fails, the error is returned",
			pageName:    "test.tmpl",
			partialsFor: partialsFor(t, "test.tmpl"),
			loadPage: func(tmplMock *templateLoaderMock) {
				page("test.tmpl", "Hello, world!", tmplMock)
			},
			createTemplate: func(base *template.Template, files []passepartout.FileWithContent) (*template.Template, error) {
				return nil, errors.New("uh-oh create template error")
			},
			expect: errContains(`failed to create template for page "test.tmpl": uh-oh create template error`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockTmplt := new(templateLoaderMock)
			mockTmplt.Test(t)
			tc.loadPage(mockTmplt)
			loader := passepartout.Loader{
				PartialsFor:    tc.partialsFor,
				TemplateLoader: mockTmplt,
				CreateTemplate: tc.createTemplate,
			}

			actual, err := loader.Page(tc.pageName)

			tc.expect(t, actual, err)
		})
	}
}

func TestLoader_PageInLayout(t *testing.T) {
	for _, tc := range []struct {
		name           string
		pageName       string
		layoutName     string
		partialsFor    func(string) ([]passepartout.FileWithContent, error)
		loadPage       func(tmplMock *templateLoaderMock)
		createTemplate func(base *template.Template, files []passepartout.FileWithContent) (*template.Template, error)
		expect         func(t *testing.T, actual *template.Template, err error)
	}{
		{
			name:        "with no errors and referencing a partial a useful template is returned",
			pageName:    "test.tmpl",
			layoutName:  "layouts/default.tmpl",
			partialsFor: partialsFor(t, "test.tmpl", passepartout.FileWithContent{Name: "_example.tmpl", Content: "- an example partial!"}),
			loadPage: func(tmplMock *templateLoaderMock) {
				pageInLayout(
					"test.tmpl",
					"layouts/default.tmpl",
					tmplMock,
					passepartout.FileWithContent{Name: "layouts/default.tmpl", Content: `HEADER {% define "content" %}CONTENT{% end %} FOOTER`},
					passepartout.FileWithContent{Name: "test.tmpl", Content: "Hello, world!"},
				)
			},
			createTemplate: createTemplate(
				t,
				nil,
				[]passepartout.FileWithContent{
					{Name: "_example.tmpl", Content: "- an example partial!"},
					{Name: "layouts/default.tmpl", Content: `HEADER {% define "content" %}CONTENT{% end %} FOOTER`},
					{Name: "test.tmpl", Content: `Hello, world!`},
				},
				template.Must(template.New("test.tmpl").Parse("Greetings layouted world!")),
			),
			expect: func(t *testing.T, actual *template.Template, err error) {
				require.NoError(t, err)
				buf := new(bytes.Buffer)
				require.NoError(t, actual.Execute(buf, nil), "expected to be able to execute the returned template")
				require.Equal(t, "Greetings layouted world!", buf.String())
			},
		},
		{
			name:       "when loading partials fails, the error is returned",
			pageName:   "test.tmpl",
			layoutName: "layouts/default.tmpl",
			partialsFor: func(page string) ([]passepartout.FileWithContent, error) {
				return nil, errors.New("uh-oh partial error")
			},
			loadPage:       func(tmplMock *templateLoaderMock) {},
			createTemplate: noTemplate,
			expect:         errContains(`failed to collect partials for page "test.tmpl": uh-oh partial error`),
		},
		{
			name:        "when loading the template fails, the error is returned",
			pageName:    "test.tmpl",
			layoutName:  "layouts/default.tmpl",
			partialsFor: partialsFor(t, "test.tmpl"),
			loadPage: func(tmplMock *templateLoaderMock) {
				tmplMock.On("PageInLayout", "test.tmpl", "layouts/default.tmpl").
					Return([]passepartout.FileWithContent(nil), errors.New("uh-oh template error"))
			},
			createTemplate: noTemplate,
			expect:         errContains(`failed to collect all page "test.tmpl" in layout "layouts/default.tmpl": uh-oh template error`),
		},
		{
			name:        "when creating the template fails, the error is returned",
			pageName:    "test.tmpl",
			layoutName:  "layouts/default.tmpl",
			partialsFor: partialsFor(t, "test.tmpl"),
			loadPage: func(tmplMock *templateLoaderMock) {
				pageInLayout(
					"test.tmpl",
					"layouts/default.tmpl",
					tmplMock,
					passepartout.FileWithContent{Name: "layouts/default.tmpl", Content: `HEADER {% define "content" %}CONTENT{% end %} FOOTER`},
					passepartout.FileWithContent{Name: "test.tmpl", Content: "Hello, world!"},
				)
			},
			createTemplate: func(base *template.Template, files []passepartout.FileWithContent) (*template.Template, error) {
				return nil, errors.New("uh-oh create template error")
			},
			expect: errContains(`failed to create template for page "test.tmpl" in layout "layouts/default.tmpl": uh-oh create template error`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockTmplt := new(templateLoaderMock)
			mockTmplt.Test(t)
			tc.loadPage(mockTmplt)
			loader := passepartout.Loader{
				PartialsFor:    tc.partialsFor,
				TemplateLoader: mockTmplt,
				CreateTemplate: tc.createTemplate,
			}

			tmpl, err := loader.InLayout(tc.pageName, tc.layoutName)

			tc.expect(t, tmpl, err)
		})
	}
}

func TestPartialsInFolderOnly(t *testing.T) {

	for _, tc := range []struct {
		name     string
		pageName string
		fs       fstest.MapFS
		expect   func(t *testing.T, actual []passepartout.FileWithContent, err error)
	}{
		{
			name:     "returns all the files in the folder named after the name without extension",
			pageName: "test.tmpl",
			fs: fstest.MapFS{
				"test/_item.tmpl":  {Data: []byte("item partial")},
				"test/_item2.tmpl": {Data: []byte("item partial 2")},
			},
			expect: func(t *testing.T, actual []passepartout.FileWithContent, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]passepartout.FileWithContent{
						{Name: "test/_item.tmpl", Content: "item partial"},
						{Name: "test/_item2.tmpl", Content: "item partial 2"},
					},
					actual,
					"expected to have found both partials in the corresponding folder",
				)
			},
		},
		{
			// I'm doing this just to indicate that this is expected behavior,
			// I don't care about removing _all extensions_, so this is the decision and it's recorded.
			name:     "returns all the files in the folder named after the name without the last extension",
			pageName: "test.tmpl.html",
			fs: fstest.MapFS{
				"test.tmpl/_item.tmpl": {Data: []byte("item partial")},
			},
			expect: func(t *testing.T, actual []passepartout.FileWithContent, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]passepartout.FileWithContent{
						{Name: "test.tmpl/_item.tmpl", Content: "item partial"},
					},
					actual,
					"expected to have found the partial in the folder named after the template with the last extension removed",
				)
			},
		},
		{
			name:     "returns no partials when none matches partials available",
			pageName: "test.tmpl",
			fs:       fstest.MapFS{},
			expect: func(t *testing.T, actual []passepartout.FileWithContent, err error) {
				require.NoError(t, err)
				require.Equal(t, []passepartout.FileWithContent(nil), actual, "expected to have an empty list when no partials available")
			},
		},
		{
			name:     "returns no partials when they're not in the one path we expect",
			pageName: "test.tmpl",
			fs: fstest.MapFS{
				"test2/_item.tmpl":           {Data: []byte("item partial")},
				"something-else/_item2.tmpl": {Data: []byte("item partial 2")},
				"_samefolder.tmpl":           {Data: []byte("item partial 3")},
			},
			expect: func(t *testing.T, actual []passepartout.FileWithContent, err error) {
				require.NoError(t, err)
				require.Equal(t, []passepartout.FileWithContent(nil), actual, "expected to not have loaded any of the partials not in the expected folder")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			loader := passepartout.PartialsInFolderOnly{FS: tc.fs}

			actual, err := loader.Load(tc.pageName)

			tc.expect(t, actual, err)
		})
	}
}

func TestTemplateByNameLoader_Page(t *testing.T) {
	t.Run("when the page doesn't exist it returns an error", func(t *testing.T) {
		l := passepartout.TemplateByNameLoader{FS: fstest.MapFS{}}

		actual, err := l.Page("test.tmpl")

		require.ErrorContains(t, err, "failed to read template: open test.tmpl")
		require.Nil(t, actual, "expected no results when an error returned")
	})

	t.Run("when the the page exists but at a different path it returns an error", func(t *testing.T) {
		l := passepartout.TemplateByNameLoader{FS: fstest.MapFS{
			"subpage/test.tmpl": {Data: []byte("Hello")},
		}}

		actual, err := l.Page("test.tmpl")

		require.ErrorContains(t, err, "failed to read template: open test.tmpl")
		require.Nil(t, actual, "expected no results when an error returned")
	})

	t.Run("when the page exists at the exact path the name and content is returned", func(t *testing.T) {
		l := passepartout.TemplateByNameLoader{FS: fstest.MapFS{
			"test.tmpl": {Data: []byte("Hello")},
		}}

		actual, err := l.Page("test.tmpl")

		require.NoError(t, err)
		require.Equal(t, []passepartout.FileWithContent{{Name: "test.tmpl", Content: "Hello"}}, actual)
	})
}

func TestTemplateByNameLoader_PageInLayout(t *testing.T) {
	t.Run("when the page doesn't exist it returns an error", func(t *testing.T) {
		l := passepartout.TemplateByNameLoader{FS: fstest.MapFS{}}

		actual, err := l.PageInLayout("test.tmpl", "layout.tmpl")

		require.ErrorContains(t, err, "failed to read template: open test.tmpl")
		require.Nil(t, actual, "expected no results when an error returned")
	})

	t.Run("when the layout doesn't exist it returns an error", func(t *testing.T) {
		l := passepartout.TemplateByNameLoader{FS: fstest.MapFS{
			"test.tmpl": {Data: []byte("Hello")},
		}}

		actual, err := l.PageInLayout("test.tmpl", "layout.tmpl")

		require.ErrorContains(t, err, "failed to read layout template: open layout.tmpl")
		require.Nil(t, actual, "expected no results when an error returned")
	})

	t.Run("when both the page and layout exists it returns the page wrapped for use with the layout", func(t *testing.T) {
		l := passepartout.TemplateByNameLoader{FS: fstest.MapFS{
			"test.tmpl":   {Data: []byte("Hello")},
			"layout.tmpl": {Data: []byte("Layout content")},
		}}

		actual, err := l.PageInLayout("test.tmpl", "layout.tmpl")

		require.NoError(t, err)
		require.Equal(t, []passepartout.FileWithContent{
			// IMPORTANT: the layout is first so any "define"s made in the layout doesn't override ones made in subsequent templates.
			{Name: "layout.tmpl", Content: "Layout content"},
			{Name: "test.tmpl", Content: `{{ define "content" }}Hello{{ end }}`},
		}, actual)
	})
}

func TestCreateTemplate(t *testing.T) {
	t.Run("when there's a problem parsing a template an error is returned", func(t *testing.T) {
		actual, err := passepartout.CreateTemplate(nil, []passepartout.FileWithContent{{
			Name:    "invalid.tmpl",
			Content: "{{ .Missing",
		}})

		require.ErrorContains(t, err, "failed to parse template")
		require.Nil(t, actual, "expected no results when an error is returned")
	})

	t.Run("it has all the passed in files as templates", func(t *testing.T) {
		baseTemplate := template.New("base")
		files := []passepartout.FileWithContent{
			{Name: "file1.tmpl", Content: "Content 1"},
			{Name: "file2.tmpl", Content: "Content 2"},
		}

		actual, err := passepartout.CreateTemplate(baseTemplate, files)

		require.NoError(t, err)
		require.Equal(t, `; defined templates are: "file1.tmpl", "file2.tmpl"`, actual.DefinedTemplates())
	})

	t.Run("it uses the base template provided as the parent for all new created templates", func(t *testing.T) {
		baseTemplate := template.New("base").
			Funcs(template.FuncMap{"customFunc": func() string { return "custom" }})
		files := []passepartout.FileWithContent{
			// calling a custom function defined on the base template to show it's used
			{Name: "file1.tmpl", Content: "{{customFunc}}"},
		}

		actual, err := passepartout.CreateTemplate(baseTemplate, files)

		require.NoError(t, err)
		buf := new(bytes.Buffer)
		err = actual.Lookup("file1.tmpl").Execute(buf, nil)
		require.NoError(t, err, "expected to execute the template without error")
		require.Equal(t, "custom", buf.String(), "expected the base template's custom function to be available")
	})
}
