package ppdefaults_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/gaqzi/passepartout/ppdefaults"
)

func TestPartialsInFolderOnly(t *testing.T) {
	for _, tc := range []struct {
		name     string
		pageName string
		fs       fstest.MapFS
		expect   func(t *testing.T, actual []ppdefaults.FileWithContent, err error)
	}{
		{
			name:     "returns all the files in the folder named after the name without extension",
			pageName: "test.tmpl",
			fs: fstest.MapFS{
				"test/_item.tmpl":  {Data: []byte("item partial")},
				"test/_item2.tmpl": {Data: []byte("item partial 2")},
			},
			expect: func(t *testing.T, actual []ppdefaults.FileWithContent, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]ppdefaults.FileWithContent{
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
			expect: func(t *testing.T, actual []ppdefaults.FileWithContent, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]ppdefaults.FileWithContent{
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
			expect: func(t *testing.T, actual []ppdefaults.FileWithContent, err error) {
				require.NoError(t, err)
				require.Equal(t, []ppdefaults.FileWithContent(nil), actual, "expected to have an empty list when no partials available")
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
			expect: func(t *testing.T, actual []ppdefaults.FileWithContent, err error) {
				require.NoError(t, err)
				require.Equal(t, []ppdefaults.FileWithContent(nil), actual, "expected to not have loaded any of the partials not in the expected folder")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			loader := ppdefaults.PartialsInFolderOnly{FS: tc.fs}

			actual, err := loader.Load(tc.pageName)

			tc.expect(t, actual, err)
		})
	}
}

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
	} {
		t.Run(tc.name, func(t *testing.T) {
			loader := ppdefaults.PartialsWithCommon{FS: tc.fs, CommonDir: "partials"}

			actual, err := loader.Load(tc.pageName)

			tc.expect(t, actual, err)
		})
	}
}
