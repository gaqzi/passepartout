# Passepartout

Passepartout is a Go template management library that provides a structured approach to organizing and rendering HTML/text templates.

## Features

- Template hierarchy management with layouts, pages, and partials
- Consistent folder structure conventions by default, fully customizable.
- Automatic loading of partial templates
- Layout system for wrapping content in consistent structures
- Works with standard filesystem and Go's embed
- Efficient template caching

## Installation

```bash
go get github.com/gaqzi/passepartout
```

## Default Template Structure

Passepartout uses a convention-based folder structure:

```
templates/                         # base folder
templates/layouts/                 # layout templates
templates/<domain>/                # domain-specific templates
templates/<domain>/<page>.<ext>    # page templates
templates/<domain>/<page>/_<partial>.<ext>  # partial templates
```

When a page template has a folder with the same name (without extension), all partials in that folder are automatically loaded and available to the template.

Each template is named after its path (excluding the templates prefix):
- `templates/reviews/show.tmpl` is named `reviews/show.tmpl`
- `templates/reviews/show/_details.tmpl` is named `reviews/show/_details.tmpl`

### Layout Example

Layouts are templates that wrap around page content. A simple example:

**Layout template** (`layouts/base.tmpl`):
```html
<!DOCTYPE html>
<html>
<head>
    <title>Example</title>
</head>
<body>
    {{ block "content" . }}{{ end }}
</body>
</html>
```

**Standalone template** (`home/index.tmpl`):
```html
<h1>Hello, {{ .Name }}!</h1>
```

When rendered with `p.RenderInLayout(writer, "layouts/base.tmpl", "home/index.tmpl", data)`, 
the standalone content is inserted into the layout where the `content` block is defined, and the standalone template is automatically wrapped.

### Alternatives

#### PartialsWithCommon

Loads partials from a folder named after the template as well as any templates found in a common folder.

See [Advanced Configuration](#advanced-configuration) for how to configure.

#### Template configuration

You can build a new `ppdefault.Loader` which can use any `html/template` or `text/template` you want as the starting point for all templates loaded from disk. This allows you to configure that missing templates panics, to provide custom template functions, and so on.

See [Advanced Configuration](#advanced-configuration) for how to build a custom loader and give it `.TemplateConfig(tmpl)`.

## Usage

### Basic Usage

```go
package main

import (
    "os"
    "fmt"
    
    "github.com/gaqzi/passepartout"
)

func main() {
    // Initialize with filesystem and remove "templates" from the path
    p, err := passepartout.LoadFrom(os.DirFS("templates/").(passepartout.FS))
    if err != nil {
        panic(err)
    }
    
    data := map[string]any{"Items": []string{"Hello", "World"}}
    
    // Render the template standalone
    err = p.Render(os.Stdout, "index/main.tmpl", data)
    if err != nil {
        panic(err)
    }
    
    // Render the template wrapped in a layout
    err = p.RenderInLayout(os.Stdout, "layouts/main.tmpl", "index/main.tmpl", data)
    if err != nil {
        panic(err)
    }
}
```

### With Go Embed

Since passepartout uses `os.FS` to load files it will also work when you embed your templates into your Go binary.

```go
package main

import (
    "embed"
    "fmt"
    "os"
    
    "github.com/gaqzi/passepartout"
)

//go:embed templates/*
var templates embed.FS

func main() {
    // Use embedded filesystem with prefix removed
    fsys, err := passepartout.FSWithoutPrefix(templates, "templates")
    if err != nil {
        panic(err)
    }
    
    p, err := passepartout.LoadFrom(fsys)
    if err != nil {
        panic(err)
    }
    
    // Render the template standalone
    data := map[string]any{"Items": []string{"Hello", "World"}}
    err = p.Render(os.Stdout, "index/main.tmpl", data)
    if err != nil {
        panic(err)
    }
}
```

### Advanced Configuration

For more control over template loading, use the builder pattern:

```go
package main

import (
    "os"
    
    "github.com/gaqzi/passepartout"
    "github.com/gaqzi/passepartout/ppdefaults"
)

func main() {
    // Create custom loader with builder pattern
	fsys := os.DirFS("templates/")
    loader := ppdefaults.NewLoaderBuilder().
        WithDefaults(fsys).
        WithCache(true).
        WithPartials(ppdefaults.PartialsWithCommon{FS: fsys}).
        Build()
    
    p := passepartout.New(loader)
    
    // Use passepartout with custom loader
    data := map[string]any{"Name": "World"}
    err := p.Render(os.Stdout, "index/main.tmpl", data)
    if err != nil {
        panic(err)
    }
}
```

## Development

- Setup: `./script/bootstrap`
- Run tests: `./script/test`

## License

MIT
