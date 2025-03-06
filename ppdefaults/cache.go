package ppdefaults

import "sync"

type loader interface {
	Standalone(name string) ([]FileWithContent, error)
	InLayout(name, layout string) ([]FileWithContent, error)
}

type CachedLoader struct {
	loader loader
	data   *sync.Map
}

// NewCachedLoader will cache successful calls to the passed in loader and return the result on repeated calls.
// If an error is returned from the underlying loader the call will not be cached.
func NewCachedLoader(l loader) *CachedLoader {
	return &CachedLoader{loader: l, data: new(sync.Map)}
}

func (c *CachedLoader) loadOrStore(cacheKey string, load func() ([]FileWithContent, error)) ([]FileWithContent, error) {
	if v, ok := c.data.Load(cacheKey); ok {
		return v.([]FileWithContent), nil
	}

	files, err := load()
	if err != nil {
		return nil, err
	}
	c.data.Store(cacheKey, files)

	return files, nil
}

func (c *CachedLoader) Standalone(name string) ([]FileWithContent, error) {
	return c.loadOrStore(name, func() ([]FileWithContent, error) {
		return c.loader.Standalone(name)
	})
}

func (c *CachedLoader) InLayout(name, layout string) ([]FileWithContent, error) {
	return c.loadOrStore(name+"|"+layout, func() ([]FileWithContent, error) {
		return c.loader.InLayout(name, layout)
	})
}
