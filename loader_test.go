package passepartout_test

import (
	"bytes"
	"errors"
	"html/template"
	"testing"

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

func TestLoader(t *testing.T) {
	t.Run("with no errors and referencing a partial a useful template is returned", func(t *testing.T) {
		tmpl := new(templateLoaderMock)
		tmpl.Test(t)
		page("test.tmpl", "Hello, world!", tmpl)
		loader := passepartout.Loader{
			PartialsFor:    partialsFor(t, "test.tmpl", passepartout.FileWithContent{Name: "_example.tmpl", Content: "- an example partial!"}),
			TemplateLoader: tmpl,
			CreateTemplate: createTemplate(
				t,
				nil,
				[]passepartout.FileWithContent{{Name: "_example.tmpl", Content: "- an example partial!"}, {Name: "test.tmpl", Content: "Hello, world!"}},
				template.Must(template.New("test.tmpl").Parse("Greetings world!")),
			),
		}

		actual, err := loader.Page("test.tmpl")

		require.NoError(t, err)
		buf := new(bytes.Buffer)
		require.NoError(t, actual.Execute(buf, nil), "expected to be able to execute the returned template")
		require.Equal(t, "Greetings world!", buf.String())
	})

	t.Run("when loading partials fails, the error is returned", func(t *testing.T) {
		loader := passepartout.Loader{
			PartialsFor: func(page string) ([]passepartout.FileWithContent, error) {
				return nil, errors.New("uh-oh partial error")
			},
			TemplateLoader: new(templateLoaderMock),
		}

		actual, err := loader.Page("test.tmpl")

		require.ErrorContains(t, err, `failed to collect all files for page "test.tmpl": uh-oh partial error`, "expected to have wrapped the error from PartialsFor")
		require.Nil(t, actual)
	})

	t.Run("when loading the template fails, the error is returned", func(t *testing.T) {
		tmplMock := new(templateLoaderMock)
		tmplMock.Test(t)
		tmplMock.On("Page", "test.tmpl").Return([]passepartout.FileWithContent(nil), errors.New("uh-oh template error"))
		loader := passepartout.Loader{
			PartialsFor:    partialsFor(t, "test.tmpl"),
			TemplateLoader: tmplMock,
		}

		actual, err := loader.Page("test.tmpl")

		require.ErrorContains(t, err, `failed to collect all files for page "test.tmpl": uh-oh template error`, "expected to have wrapped the error from TemplateLoader.Page")
		require.Nil(t, actual)
	})

	t.Run("when creating the template fails, the error is returned", func(t *testing.T) {
		tmplMock := new(templateLoaderMock)
		tmplMock.Test(t)
		page("test.tmpl", "Hello, world!", tmplMock)
		loader := passepartout.Loader{
			PartialsFor:    partialsFor(t, "test.tmpl"),
			TemplateLoader: tmplMock,
			CreateTemplate: func(base *template.Template, files []passepartout.FileWithContent) (*template.Template, error) {
				return nil, errors.New("uh-oh create template error")
			},
		}

		actual, err := loader.Page("test.tmpl")

		require.ErrorContains(t, err, `failed to create template for page "test.tmpl": uh-oh create template error`, "expected to have wrapped the error from CreateTemplate")
		require.Nil(t, actual)
	})
}
