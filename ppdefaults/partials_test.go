package ppdefaults_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/gaqzi/passepartout/ppdefaults"
)

func TestPartialsWithCommon(t *testing.T) {
	for _, tc := range []struct {
		name     string
		pageName string
		fs       fstest.MapFS
		expect   func(t *testing.T, actual []ppdefaults.FileWithContent, err error)
	}{
		{
			name:     "returns partials from both template-specific folder and common partials folder",
			pageName: "test.tmpl",
			fs: fstest.MapFS{
				"test/_item.tmpl":       {Data: []byte("template partial")},
				"partials/_common.tmpl": {Data: []byte("common partial")},
			},
			expect: func(t *testing.T, actual []ppdefaults.FileWithContent, err error) {
				require.NoError(t, err)
				require.ElementsMatch(
					t,
					[]ppdefaults.FileWithContent{
						{Name: "test/_item.tmpl", Content: "template partial"},
						{Name: "partials/_common.tmpl", Content: "common partial"},
					},
					actual,
					"expected to have found partials from both the template-specific folder and the common partials folder",
				)
			},
		},
		{
			name:     "returns only template-specific partials when no common partials exist",
			pageName: "test.tmpl",
			fs: fstest.MapFS{
				"test/_item.tmpl": {Data: []byte("template partial")},
			},
			expect: func(t *testing.T, actual []ppdefaults.FileWithContent, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]ppdefaults.FileWithContent{
						{Name: "test/_item.tmpl", Content: "template partial"},
					},
					actual,
					"expected to have found only template-specific partials when no common partials exist",
				)
			},
		},
		{
			name:     "returns only common partials when no template-specific partials exist",
			pageName: "test.tmpl",
			fs: fstest.MapFS{
				"partials/_common.tmpl": {Data: []byte("common partial")},
			},
			expect: func(t *testing.T, actual []ppdefaults.FileWithContent, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]ppdefaults.FileWithContent{
						{Name: "partials/_common.tmpl", Content: "common partial"},
					},
					actual,
					"expected to have found only common partials when no template-specific partials exist",
				)
			},
		},
		{
			name:     "returns no partials when neither template-specific nor common partials exist",
			pageName: "test.tmpl",
			fs:       fstest.MapFS{},
			expect: func(t *testing.T, actual []ppdefaults.FileWithContent, err error) {
				require.NoError(t, err)
				require.Empty(t, actual, "expected to have an empty list when no partials available")
			},
		},
		{
			name:     "handles template with multiple extensions correctly",
			pageName: "test.tmpl.html",
			fs: fstest.MapFS{
				"test.tmpl/_item.tmpl":  {Data: []byte("template partial")},
				"partials/_common.tmpl": {Data: []byte("common partial")},
			},
			expect: func(t *testing.T, actual []ppdefaults.FileWithContent, err error) {
				require.NoError(t, err)
				require.ElementsMatch(
					t,
					[]ppdefaults.FileWithContent{
						{Name: "test.tmpl/_item.tmpl", Content: "template partial"},
						{Name: "partials/_common.tmpl", Content: "common partial"},
					},
					actual,
					"expected to handle template with multiple extensions correctly",
				)
			},
		},
		{
			name:     "handles conflicts by including both files",
			pageName: "test.tmpl",
			fs: fstest.MapFS{
				"test/_header.tmpl":     {Data: []byte("template header")},
				"partials/_header.tmpl": {Data: []byte("common header")},
			},
			expect: func(t *testing.T, actual []ppdefaults.FileWithContent, err error) {
				require.NoError(t, err)
				require.Len(t, actual, 2, "expected to include both files with the same name")
				// We don't assert the exact content here since either could be loaded first,
				// but we do ensure both are loaded
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			loader := ppdefaults.PartialsWithCommon{FS: tc.fs, CommonDir: "partials"}

			actual, err := loader.Load(tc.pageName)

			tc.expect(t, actual, err)
		})
	}
}
