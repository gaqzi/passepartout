// Code generated by go-builder-generator (https://github.com/kilianpaquier/go-builder-generator). DO NOT EDIT.

// Code generated from ppdefaults/loader.go

package ppdefaults

import "html/template"

//go:generate go run github.com/kilianpaquier/go-builder-generator/cmd/go-builder-generator@latest generate -d . -f loader.go -s Loader

// LoaderBuilder represents Loader's builder.
type LoaderBuilder struct {
	build Loader
}

// NewLoaderBuilder creates a new LoaderBuilder.
func NewLoaderBuilder() *LoaderBuilder {
	return &LoaderBuilder{}
}

// Copy reassigns the builder struct (behind pointer) to a new pointer and returns it.
func (b *LoaderBuilder) Copy() *LoaderBuilder {
	return &LoaderBuilder{b.build}
}

// Build returns built Loader.
func (b *LoaderBuilder) Build() *Loader {
	result := b.build
	return &result
}

// CreateTemplate sets Loader's CreateTemplate.
func (b *LoaderBuilder) CreateTemplate(createTemplate Templater) *LoaderBuilder {
	b.build.CreateTemplate = createTemplate
	return b
}

// PartialsFor sets Loader's PartialsFor.
func (b *LoaderBuilder) PartialsFor(partialsFor PartialLoader) *LoaderBuilder {
	b.build.PartialsFor = partialsFor
	return b
}

// TemplateConfig sets Loader's TemplateConfig.
func (b *LoaderBuilder) TemplateConfig(templateConfig template.Template) *LoaderBuilder {
	b.build.TemplateConfig = &templateConfig
	return b
}

// TemplateLoader sets Loader's TemplateLoader.
func (b *LoaderBuilder) TemplateLoader(templateLoader TemplateLoader) *LoaderBuilder {
	b.build.TemplateLoader = templateLoader
	return b
}
