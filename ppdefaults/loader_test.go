package ppdefaults_test

import (
	"bytes"
	"errors"
	"html/template"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/gaqzi/passepartout/ppdefaults"
)

type templateLoaderMock struct {
	mock.Mock
}

func (t *templateLoaderMock) Standalone(name string) ([]ppdefaults.FileWithContent, error) {
	args := t.Called(name)
	return args.Get(0).([]ppdefaults.FileWithContent), args.Error(1)
}

func (t *templateLoaderMock) InLayout(name string, layout string) ([]ppdefaults.FileWithContent, error) {
	args := t.Called(name, layout)
	return args.Get(0).([]ppdefaults.FileWithContent), args.Error(1)
}

func standalone(name, content string, t *templateLoaderMock) {
	t.
		On("Standalone", name).
		Return([]ppdefaults.FileWithContent{{Name: name, Content: content}}, nil)
}

func inLayout(page, layout string, t *templateLoaderMock, templates ...ppdefaults.FileWithContent) {

	t.
		On("InLayout", page, layout).
		Return(templates, nil)
}

func partialsFor(t *testing.T, name string, files ...ppdefaults.FileWithContent) func(string) ([]ppdefaults.FileWithContent, error) {
	t.Helper()
	return func(page string) ([]ppdefaults.FileWithContent, error) {
		t.Helper()
		require.Equal(t, name, page, "expected to have called PartialsFor with the name passed to Standalone")

		return files, nil
	}
}

func createTemplate(t *testing.T, base *template.Template, files []ppdefaults.FileWithContent, tmpl *template.Template) func(base *template.Template, files []ppdefaults.FileWithContent) (*template.Template, error) {
	t.Helper()
	return func(inBase *template.Template, inFiles []ppdefaults.FileWithContent) (*template.Template, error) {
		require.Equal(t, base, inBase, "expected to have called CreateTemplate with the base template")
		require.Equal(t, files, inFiles, "expected to have called CreateTemplate with the files from PartialsFor and TemplateLoader.Standalone")

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

func noTemplate(base *template.Template, files []ppdefaults.FileWithContent) (*template.Template, error) {
	return nil, nil
}

func TestLoader_Standalone(t *testing.T) {
	for _, tc := range []struct {
		name           string
		pageName       string
		partialsFor    func(string) ([]ppdefaults.FileWithContent, error)
		loadPage       func(tmplMock *templateLoaderMock)
		createTemplate func(base *template.Template, files []ppdefaults.FileWithContent) (*template.Template, error)
		expect         func(t *testing.T, actual *template.Template, err error)
	}{
		{
			name:        "with no errors and referencing a partial a useful template is returned",
			pageName:    "test.tmpl",
			partialsFor: partialsFor(t, "test.tmpl", ppdefaults.FileWithContent{Name: "_example.tmpl", Content: "- an example partial!"}),
			loadPage:    func(tmplMock *templateLoaderMock) { standalone("test.tmpl", "Hello, world!", tmplMock) },
			createTemplate: createTemplate(
				t,
				nil,
				[]ppdefaults.FileWithContent{{Name: "_example.tmpl", Content: "- an example partial!"}, {Name: "test.tmpl", Content: "Hello, world!"}},
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
			partialsFor: func(page string) ([]ppdefaults.FileWithContent, error) {
				return nil, errors.New("uh-oh partial error")
			},
			loadPage:       func(tmplMock *templateLoaderMock) {},
			createTemplate: noTemplate,
			expect:         errContains(`failed to collect all files for "test.tmpl": uh-oh partial error`),
		},
		{
			name:        "when loading the template fails, the error is returned",
			pageName:    "test.tmpl",
			partialsFor: partialsFor(t, "test.tmpl"),
			loadPage: func(tmplMock *templateLoaderMock) {
				tmplMock.On("Standalone", "test.tmpl").
					Return([]ppdefaults.FileWithContent(nil), errors.New("uh-oh template error"))
			},
			createTemplate: noTemplate,
			expect:         errContains(`failed to collect all files for "test.tmpl": uh-oh template error`),
		},
		{
			name:        "when creating the template fails, the error is returned",
			pageName:    "test.tmpl",
			partialsFor: partialsFor(t, "test.tmpl"),
			loadPage: func(tmplMock *templateLoaderMock) {
				standalone("test.tmpl", "Hello, world!", tmplMock)
			},
			createTemplate: func(base *template.Template, files []ppdefaults.FileWithContent) (*template.Template, error) {
				return nil, errors.New("uh-oh create template error")
			},
			expect: errContains(`failed to create template for "test.tmpl": uh-oh create template error`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockTmplt := new(templateLoaderMock)
			mockTmplt.Test(t)
			tc.loadPage(mockTmplt)
			loader := ppdefaults.Loader{
				PartialsFor:    tc.partialsFor,
				TemplateLoader: mockTmplt,
				CreateTemplate: tc.createTemplate,
			}

			actual, err := loader.Standalone(tc.pageName)

			tc.expect(t, actual, err)
		})
	}
}

func TestLoader_InLayout(t *testing.T) {
	for _, tc := range []struct {
		name           string
		pageName       string
		layoutName     string
		partialsFor    func(string) ([]ppdefaults.FileWithContent, error)
		loadPage       func(tmplMock *templateLoaderMock)
		createTemplate func(base *template.Template, files []ppdefaults.FileWithContent) (*template.Template, error)
		expect         func(t *testing.T, actual *template.Template, err error)
	}{
		{
			name:        "with no errors and referencing a partial a useful template is returned",
			pageName:    "test.tmpl",
			layoutName:  "layouts/default.tmpl",
			partialsFor: partialsFor(t, "test.tmpl", ppdefaults.FileWithContent{Name: "_example.tmpl", Content: "- an example partial!"}),
			loadPage: func(tmplMock *templateLoaderMock) {
				inLayout(
					"test.tmpl",
					"layouts/default.tmpl",
					tmplMock,
					ppdefaults.FileWithContent{Name: "layouts/default.tmpl", Content: `HEADER {% define "content" %}CONTENT{% end %} FOOTER`},
					ppdefaults.FileWithContent{Name: "test.tmpl", Content: "Hello, world!"},
				)
			},
			createTemplate: createTemplate(
				t,
				nil,
				[]ppdefaults.FileWithContent{
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
			partialsFor: func(page string) ([]ppdefaults.FileWithContent, error) {
				return nil, errors.New("uh-oh partial error")
			},
			loadPage:       func(tmplMock *templateLoaderMock) {},
			createTemplate: noTemplate,
			expect:         errContains(`failed to collect partials for "test.tmpl": uh-oh partial error`),
		},
		{
			name:        "when loading the template fails, the error is returned",
			pageName:    "test.tmpl",
			layoutName:  "layouts/default.tmpl",
			partialsFor: partialsFor(t, "test.tmpl"),
			loadPage: func(tmplMock *templateLoaderMock) {
				tmplMock.On("InLayout", "test.tmpl", "layouts/default.tmpl").
					Return([]ppdefaults.FileWithContent(nil), errors.New("uh-oh template error"))
			},
			createTemplate: noTemplate,
			expect:         errContains(`failed to collect all for "test.tmpl" in layout "layouts/default.tmpl": uh-oh template error`),
		},
		{
			name:        "when creating the template fails, the error is returned",
			pageName:    "test.tmpl",
			layoutName:  "layouts/default.tmpl",
			partialsFor: partialsFor(t, "test.tmpl"),
			loadPage: func(tmplMock *templateLoaderMock) {
				inLayout(
					"test.tmpl",
					"layouts/default.tmpl",
					tmplMock,
					ppdefaults.FileWithContent{Name: "layouts/default.tmpl", Content: `HEADER {% define "content" %}CONTENT{% end %} FOOTER`},
					ppdefaults.FileWithContent{Name: "test.tmpl", Content: "Hello, world!"},
				)
			},
			createTemplate: func(base *template.Template, files []ppdefaults.FileWithContent) (*template.Template, error) {
				return nil, errors.New("uh-oh create template error")
			},
			expect: errContains(`failed to create template for "test.tmpl" in layout "layouts/default.tmpl": uh-oh create template error`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockTmplt := new(templateLoaderMock)
			mockTmplt.Test(t)
			tc.loadPage(mockTmplt)
			loader := ppdefaults.Loader{
				PartialsFor:    tc.partialsFor,
				TemplateLoader: mockTmplt,
				CreateTemplate: tc.createTemplate,
			}

			tmpl, err := loader.InLayout(tc.pageName, tc.layoutName)

			tc.expect(t, tmpl, err)
		})
	}
}

func TestLoader_TemplateConfig(t *testing.T) {
	t.Run("in Standalone is passed into CreateTemplate on use", func(t *testing.T) {
		mockTmplt := new(templateLoaderMock)
		mockTmplt.Test(t)
		standalone("test.tmpl", "Hello, world!", mockTmplt)
		expectedTemplate := template.Must(template.New("template config").Parse("yohooo~!"))
		loader := ppdefaults.Loader{
			TemplateConfig: expectedTemplate,
			PartialsFor:    partialsFor(t, "test.tmpl"),
			TemplateLoader: mockTmplt,
			CreateTemplate: func(base *template.Template, files []ppdefaults.FileWithContent) (*template.Template, error) {
				require.Equal(t, expectedTemplate, base, "expected to have received the configured expected template when creating templates")
				return nil, nil
			},
		}

		_, err := loader.Standalone("test.tmpl")

		require.NoError(t, err, "expected to have no error and that any assertions are happening inside [CreateTemplate]")
	})

	t.Run("in InLayout is passed into CreateTemplate on use", func(t *testing.T) {
		mockTmplt := new(templateLoaderMock)
		mockTmplt.Test(t)
		inLayout(
			"test.tmpl",
			"layouts/default.tmpl",
			mockTmplt,
			ppdefaults.FileWithContent{Name: "layouts/default.tmpl", Content: `HEADER {% define "content" %}CONTENT{% end %} FOOTER`},
			ppdefaults.FileWithContent{Name: "test.tmpl", Content: "Hello, world!"},
		)
		expectedTemplate := template.Must(template.New("template config").Parse("yohooo~!"))
		loader := ppdefaults.Loader{
			TemplateConfig: expectedTemplate,
			PartialsFor:    partialsFor(t, "test.tmpl"),
			TemplateLoader: mockTmplt,
			CreateTemplate: func(base *template.Template, files []ppdefaults.FileWithContent) (*template.Template, error) {
				require.Equal(t, expectedTemplate, base, "expected to have received the configured expected template when creating templates")
				return nil, nil
			},
		}

		_, err := loader.InLayout("test.tmpl", "layouts/default.tmpl")

		require.NoError(t, err, "expected to have no error and that any assertions are happening inside [CreateTemplate]")
	})
}

func TestTemplateByNameLoader_Standalone(t *testing.T) {
	t.Run("when the file doesn't exist it returns an error", func(t *testing.T) {
		l := ppdefaults.TemplateByNameLoader{FS: fstest.MapFS{}}

		actual, err := l.Standalone("test.tmpl")

		require.ErrorContains(t, err, "failed to read template: open test.tmpl")
		require.Nil(t, actual, "expected no results when an error returned")
	})

	t.Run("when the the file exists but at a different path it returns an error", func(t *testing.T) {
		l := ppdefaults.TemplateByNameLoader{FS: fstest.MapFS{
			"subpage/test.tmpl": {Data: []byte("Hello")},
		}}

		actual, err := l.Standalone("test.tmpl")

		require.ErrorContains(t, err, "failed to read template: open test.tmpl")
		require.Nil(t, actual, "expected no results when an error returned")
	})

	t.Run("when the file exists at the exact path the name and content is returned", func(t *testing.T) {
		l := ppdefaults.TemplateByNameLoader{FS: fstest.MapFS{
			"test.tmpl": {Data: []byte("Hello")},
		}}

		actual, err := l.Standalone("test.tmpl")

		require.NoError(t, err)
		require.Equal(t, []ppdefaults.FileWithContent{{Name: "test.tmpl", Content: "Hello"}}, actual)
	})
}

func TestTemplateByNameLoader_InLayout(t *testing.T) {
	t.Run("when the file doesn't exist it returns an error", func(t *testing.T) {
		l := ppdefaults.TemplateByNameLoader{FS: fstest.MapFS{}}

		actual, err := l.InLayout("test.tmpl", "layout.tmpl")

		require.ErrorContains(t, err, "failed to read template: open test.tmpl")
		require.Nil(t, actual, "expected no results when an error returned")
	})

	t.Run("when the layout doesn't exist it returns an error", func(t *testing.T) {
		l := ppdefaults.TemplateByNameLoader{FS: fstest.MapFS{
			"test.tmpl": {Data: []byte("Hello")},
		}}

		actual, err := l.InLayout("test.tmpl", "layout.tmpl")

		require.ErrorContains(t, err, "failed to read layout template: open layout.tmpl")
		require.Nil(t, actual, "expected no results when an error returned")
	})

	t.Run("when both the file and layout exists it returns the template wrapped for use within the layout", func(t *testing.T) {
		l := ppdefaults.TemplateByNameLoader{FS: fstest.MapFS{
			"test.tmpl":   {Data: []byte("Hello")},
			"layout.tmpl": {Data: []byte("Layout content")},
		}}

		actual, err := l.InLayout("test.tmpl", "layout.tmpl")

		require.NoError(t, err)
		require.Equal(t, []ppdefaults.FileWithContent{
			// IMPORTANT: the layout is first so any "define"s made in the layout doesn't override ones made in subsequent templates.
			{Name: "layout.tmpl", Content: "Layout content"},
			{Name: "test.tmpl", Content: `{{ define "content" }}Hello{{ end }}`},
		}, actual)
	})
}

func TestCreateTemplate(t *testing.T) {
	t.Run("when there's a problem parsing a template an error is returned", func(t *testing.T) {
		actual, err := ppdefaults.CreateTemplate(nil, []ppdefaults.FileWithContent{{
			Name:    "invalid.tmpl",
			Content: "{{ .Missing",
		}})

		require.ErrorContains(t, err, "failed to parse template")
		require.Nil(t, actual, "expected no results when an error is returned")
	})

	t.Run("it has all the passed in files as templates", func(t *testing.T) {
		baseTemplate := template.New("base")
		files := []ppdefaults.FileWithContent{
			{Name: "file1.tmpl", Content: "Content 1"},
			{Name: "file2.tmpl", Content: "Content 2"},
		}

		actual, err := ppdefaults.CreateTemplate(baseTemplate, files)

		require.NoError(t, err)
		_, after, found := strings.Cut(actual.DefinedTemplates(), ": ")
		require.True(t, found, "expected to have created multiple templates")
		templates := strings.Split(after, ", ")
		require.ElementsMatch(t, []string{`"file1.tmpl"`, `"file2.tmpl"`}, templates)
	})

	t.Run("it uses the base template provided as the parent for all new created templates", func(t *testing.T) {
		baseTemplate := template.New("base").
			Funcs(template.FuncMap{"customFunc": func() string { return "custom" }})
		files := []ppdefaults.FileWithContent{
			// calling a custom function defined on the base template to show it's used
			{Name: "file1.tmpl", Content: "{{customFunc}}"},
		}

		actual, err := ppdefaults.CreateTemplate(baseTemplate, files)

		require.NoError(t, err)
		buf := new(bytes.Buffer)
		err = actual.Lookup("file1.tmpl").Execute(buf, nil)
		require.NoError(t, err, "expected to execute the template without error")
		require.Equal(t, "custom", buf.String(), "expected the base template's custom function to be available")
	})
}
