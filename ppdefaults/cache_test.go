package ppdefaults_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/gaqzi/passepartout/ppdefaults"
)

type mockLoader struct {
	mock.Mock
}

func (m *mockLoader) Standalone(name string) ([]ppdefaults.FileWithContent, error) {
	called := m.Called(name)

	return called.Get(0).([]ppdefaults.FileWithContent), called.Error(1)
}

func (m *mockLoader) InLayout(name, layout string) ([]ppdefaults.FileWithContent, error) {
	called := m.Called(name, layout)

	return called.Get(0).([]ppdefaults.FileWithContent), called.Error(1)
}

func TestCachedLoader(t *testing.T) {
	for _, tc := range []struct {
		name       string
		loaderCall func(loader *mockLoader)
		call       func(t *testing.T, cache *ppdefaults.CachedLoader)
	}{
		{
			name: "Standalone only calls the underlying loader once",
			loaderCall: func(loader *mockLoader) {
				loader.On("Standalone", "example.tmpl").
					Return([]ppdefaults.FileWithContent{{Name: "example.tmpl"}}, nil).
					Once()
			},
			call: func(t *testing.T, cache *ppdefaults.CachedLoader) {
				actual, err := cache.Standalone("example.tmpl")

				require.NoError(t, err)
				require.Equal(t, []ppdefaults.FileWithContent{{Name: "example.tmpl"}}, actual)
			},
		},
		{
			name: "Standalone returns an error if the underlying loader returns an error, without caching it",
			loaderCall: func(loader *mockLoader) {
				loader.On("Standalone", "example.tmpl").
					Return(([]ppdefaults.FileWithContent)(nil), errors.New("uh-oh"))
			},
			call: func(t *testing.T, cache *ppdefaults.CachedLoader) {
				actual, err := cache.Standalone("example.tmpl")

				require.ErrorContains(t, err, "uh-oh")
				require.Nil(t, actual)
			},
		},
		{
			name: "InLayout only calls the underlying loader once",
			loaderCall: func(loader *mockLoader) {
				loader.On("InLayout", "example.tmpl", "layout.tmpl").
					Return([]ppdefaults.FileWithContent{{Name: "example.tmpl"}}, nil).
					Once()
			},
			call: func(t *testing.T, cache *ppdefaults.CachedLoader) {
				actual, err := cache.InLayout("example.tmpl", "layout.tmpl")

				require.NoError(t, err)
				require.Equal(t, []ppdefaults.FileWithContent{{Name: "example.tmpl"}}, actual)
			},
		},
		{
			name: "returns an error if the underlying loader returns an error, without caching it",
			loaderCall: func(loader *mockLoader) {
				loader.On("InLayout", "example.tmpl", "layout.tmpl").
					Return(([]ppdefaults.FileWithContent)(nil), errors.New("uh-oh"))
			},
			call: func(t *testing.T, cache *ppdefaults.CachedLoader) {
				actual, err := cache.InLayout("example.tmpl", "layout.tmpl")

				require.ErrorContains(t, err, "uh-oh")
				require.Nil(t, actual)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			loader := new(mockLoader)
			loader.Test(t)
			tc.loaderCall(loader)
			cache := ppdefaults.NewCachedLoader(loader)

			// Because the core of the logic about caching, i.e. repeated calls,
			// we'll always call it twice and assume that we have an assertion set on the mock
			// when we care about the number of calls.
			for range 2 {
				tc.call(t, cache)
			}

			loader.AssertExpectations(t)
		})
	}
}
