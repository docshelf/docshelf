package disk

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
)

// A Store implements the skribe FileStore interface. It manages file storage on the local disk.
type Store struct {
	Root string
}

// New returns a new Store struct based on the given rootPath.
func New(rootPath string) Store {
	return Store{rootPath}
}

// ReadFile reads the content from an existing file on disk.
func (s Store) ReadFile(path string) ([]byte, error) {
	content, err := ioutil.ReadFile(s.fullPath(path))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read file")
	}

	return content, nil
}

// WriteFile creates or overwrites a file on disk at the given path with the given content.
func (s Store) WriteFile(path string, content []byte) error {
	return errors.Wrap(ioutil.WriteFile(s.fullPath(path), content, 0777), "failed to write file")
}

// RemoveFile removes an existing file from disk.
func (s Store) RemoveFile(path string) error {
	return errors.Wrap(os.Remove(s.fullPath(path)), "failed to remove file")
}

// ListDir returns a listing of all files that exist within a directory.
func (s Store) ListDir(path string) ([]string, error) {
	p := s.fullPath(path)
	info, err := os.Stat(p)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		files, err := ioutil.ReadDir(p)
		if err != nil {
			return nil, err
		}

		listing := make([]string, len(files))
		for _, f := range files {
			listing = append(listing, f.Name())
		}

		return listing, nil
	}

	return nil, errors.New("path is not a directory")
}

func (s Store) fullPath(path string) string {
	return fmt.Sprintf("%s/%s", s.Root, path)
}
