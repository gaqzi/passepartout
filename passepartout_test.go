package passepartout_test

import (
	"bytes"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"passepartout"
)

func noError(t *testing.T, err error) {
	t.Helper()
	require.NoError(t, err)
}

type call struct {
	name string
	data any
}

func TestPassepartout_Render(t *testing.T) {
	testCases := []struct {
		name        string
		fs          fstest.MapFS
		render      call
		expected    string
		expectError func(t *testing.T, err error)
	}{
		{
			name: "A single template with no partials or layout available and renderable",
			fs: fstest.MapFS{
				"templates/index.tmpl": {Data: []byte("body")},
			},
			render:      call{`templates/index.tmpl`, nil},
			expected:    `body`,
			expectError: noError,
		},
		{
			name: "A template with a partial in a subfolder named after the page is renderable",
			fs: fstest.MapFS{
				"templates/index.tmpl":       {Data: []byte("body\n {{ template \"templates/index/_item.tmpl\" . }}")},
				"templates/index/_item.tmpl": {Data: []byte("item partial")},
			},
			render:      call{`templates/index.tmpl`, nil},
			expected:    "body\n item partial",
			expectError: noError,
		},
		{
			name: "A template with a partial in a different subfolder than the page is not renderable",
			fs: fstest.MapFS{
				"templates/index.tmpl":      {Data: []byte("body\n {{ template \"_item.tmpl\" . }}")},
				"templates/show/_item.tmpl": {Data: []byte("item partial")},
			},
			render:   call{`templates/index.tmpl`, nil},
			expected: "",
			expectError: func(t *testing.T, err error) {
				// XXX: should there be a custom error message when the filename starts with "_" to say that we only look for partials in the current folder or sub folder named the same as the template minus extension?
				require.ErrorContains(t, err, `no such template "_item.tmpl"`, "expected a warning that the partial was not found")
			},
		},
		{
			name:     "When the page doesn't exist we get an error",
			fs:       fstest.MapFS{},
			render:   call{`templates/index.tmpl`, nil},
			expected: "",
			expectError: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `failed to read template: open templates/index.tmpl`, "expected a warning that the template was not found")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pp, err := passepartout.Load(tc.fs)
			require.NoError(t, err, "expected to have loaded all the templates without issues")

			output := bytes.NewBuffer(nil)
			err = pp.Render(output, tc.render.name, tc.render.data)

			tc.expectError(t, err)
			require.Equal(t, tc.expected, output.String())
		})
	}
}

type layoutCall struct {
	layout string
	name   string
	data   any
}

func TestPassepartout_RenderInTemplate(t *testing.T) {
	testCases := []struct {
		name        string
		fs          fstest.MapFS
		render      layoutCall
		expected    string
		expectError func(t *testing.T, err error)
	}{
		{
			name: "When there's a layout, then the template renders within the layout by wrapping the content of the template in a defined content block",
			fs: fstest.MapFS{
				"templates/layouts/default.tmpl": {Data: []byte("HEAD\n {{ block \"content\" . }}DEFAULT CONTENT{{ end }} \nFOOT")},
				"templates/index.tmpl":           {Data: []byte("body\n {{ template \"templates/index/_item.tmpl\" . }}")},
				"templates/index/_item.tmpl":     {Data: []byte("item partial")},
			},
			render:      layoutCall{`templates/layouts/default.tmpl`, `templates/index.tmpl`, nil},
			expected:    "HEAD\n body\n item partial \nFOOT",
			expectError: noError,
		},
		{
			name: "When there's multple layouts, then the template is available under both names",
			fs: fstest.MapFS{
				"templates/layouts/default.tmpl":   {Data: []byte("HEAD\n {{ block \"content\" . }}DEFAULT CONTENT{{ end }} \nFOOT")},
				"templates/layouts/secondary.tmpl": {Data: []byte("HEADER\n {{ block \"content\" . }}DEFAULT CONTENT{{ end }} \nFOOTER")},
				"templates/index.tmpl":             {Data: []byte("body\n {{ template \"templates/index/_item.tmpl\" . }}")},
				"templates/index/_item.tmpl":       {Data: []byte("item partial")},
			},
			render:      layoutCall{`templates/layouts/secondary.tmpl`, `templates/index.tmpl`, nil},
			expected:    "HEADER\n body\n item partial \nFOOTER",
			expectError: noError,
		},
		{
			name:     "When the page doesn't exist we get an error",
			fs:       fstest.MapFS{},
			render:   layoutCall{`templates/layouts/default.tmpl`, `templates/index.tmpl`, nil},
			expected: "",
			expectError: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `failed to read template: open templates/index.tmpl`, "expected a warning that the template was not found")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pp, err := passepartout.Load(tc.fs)
			require.NoError(t, err, "expected to have loaded all the templates without issues")

			output := bytes.NewBuffer(nil)
			err = pp.RenderInLayout(output, tc.render.layout, tc.render.name, tc.render.data)

			tc.expectError(t, err)
			require.Equal(t, tc.expected, output.String())
		})
	}
}
