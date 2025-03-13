package ppdefaults

import (
	"errors"
	"io/fs"
	"path"
	"strings"
)

// PartialsWithCommon loads partials in the same way as [PartialsInFolderOnly]
// and from a CommonDir, for example "partials".
type PartialsWithCommon struct {
	FS        fs.ReadDirFS
	CommonDir string
}

// Load implements the PartialLoader interface.
func (p *PartialsWithCommon) Load(page string) ([]FileWithContent, error) {
	var files []FileWithContent

	ext := path.Ext(page)
	dirName := strings.TrimSuffix(page, ext)

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
