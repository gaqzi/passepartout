package ppdefaults

import (
	"errors"
	"io/fs"
	"path"
	"strings"
)

// PartialsInFolderOnly implements the [PartialLoader] interface.
type PartialsInFolderOnly struct {
	FS fs.ReadDirFS
}

// Load gets files from a folder named after the passed in template and treats them as partials.
// Ex: a template named "something/hello.tmpl" will load any files in the folder "something/hello/".
func (p *PartialsInFolderOnly) Load(name string) ([]FileWithContent, error) {
	ext := path.Ext(name)
	dirName := strings.TrimSuffix(name, ext)

	var files []FileWithContent
	err := fs.WalkDir(p.FS, dirName, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}

		if entry.IsDir() {
			return nil
		}

		content, err := fs.ReadFile(p.FS, filePath)
		if err != nil {
			return err
		}

		files = append(files, FileWithContent{Name: filePath, Content: string(content)})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

// PartialsWithCommon implements the [PartialLoader] interface.
type PartialsWithCommon struct {
	FS        fs.ReadDirFS
	CommonDir string
}

// Load partials in the same way as [PartialsInFolderOnly.Load] and from a CommonDir, for example "partials".
func (p *PartialsWithCommon) Load(name string) ([]FileWithContent, error) {
	var files []FileWithContent

	ext := path.Ext(name)
	dirName := strings.TrimSuffix(name, ext)

	for _, dir := range []string{dirName, p.CommonDir} {
		err := fs.WalkDir(p.FS, dir, func(filePath string, entry fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}

			if entry.IsDir() {
				return nil
			}

			content, err := fs.ReadFile(p.FS, filePath)
			if err != nil {
				return err
			}

			files = append(files, FileWithContent{Name: filePath, Content: string(content)})

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}
