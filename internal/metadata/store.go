package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hocicopollo02/sandbox/internal/model"
)

var ErrNotFound = errors.New("sandbox metadata not found")

type Paths struct {
	DataRoot string
	Metadata string
	Homes    string
}

func PathsFor(home string) Paths {
	dataRoot := filepath.Join(home, ".local", "share", "sandbox")
	return Paths{
		DataRoot: dataRoot,
		Metadata: filepath.Join(dataRoot, "sandboxes"),
		Homes:    filepath.Join(dataRoot, "homes"),
	}
}

type Store struct {
	Paths Paths
}

func NewStore(paths Paths) *Store { return &Store{Paths: paths} }

func (s *Store) file(name string) string {
	return filepath.Join(s.Paths.Metadata, name+".json")
}

func (s *Store) Get(name string) (model.Record, error) {
	data, err := os.ReadFile(s.file(name))
	if errors.Is(err, os.ErrNotExist) {
		return model.Record{}, ErrNotFound
	}
	if err != nil {
		return model.Record{}, fmt.Errorf("read metadata for %q: %w", name, err)
	}
	var record model.Record
	if err := json.Unmarshal(data, &record); err != nil {
		return model.Record{}, fmt.Errorf("invalid metadata for %q: %w", name, err)
	}
	return record, nil
}

func (s *Store) Save(record model.Record) error {
	if _, err := model.ValidateName(record.Name); err != nil {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}
	if err := os.MkdirAll(s.Paths.Metadata, 0700); err != nil {
		return fmt.Errorf("create metadata directory: %w", err)
	}
	tmp, err := os.CreateTemp(s.Paths.Metadata, ".sandbox-*.json")
	if err != nil {
		return fmt.Errorf("create metadata temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("protect metadata: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write metadata: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close metadata: %w", err)
	}
	if err := os.Rename(tmpName, s.file(record.Name)); err != nil {
		return fmt.Errorf("commit metadata: %w", err)
	}
	return nil
}

func (s *Store) Delete(name string) error {
	if err := os.Remove(s.file(name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete metadata for %q: %w", name, err)
	}
	return nil
}

func (s *Store) List() ([]model.Record, error) {
	entries, err := os.ReadDir(s.Paths.Metadata)
	if errors.Is(err, os.ErrNotExist) {
		return []model.Record{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read metadata directory: %w", err)
	}
	records := make([]model.Record, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		record, err := s.Get(entry.Name()[:len(entry.Name())-len(filepath.Ext(entry.Name()))])
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func (s *Store) Home(name string) string {
	return filepath.Join(s.Paths.Homes, name)
}

// ValidateHomesRoot rejects symlinked or non-directory ancestors before a home
// is created or passed to the container runtime.
// ponytail: same-user TOCTOU remains; use a named Podman volume or VM for adversarial host isolation.
func (s *Store) ValidateHomesRoot() error {
	root, err := filepath.Abs(s.Paths.Homes)
	if err != nil {
		return fmt.Errorf("resolve managed homes directory: %w", err)
	}
	for current := root; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		switch {
		case err == nil:
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("refusing symlinked managed homes directory: %s", current)
			}
			if !info.IsDir() {
				return fmt.Errorf("managed homes path is not a directory: %s", current)
			}
		case !errors.Is(err, os.ErrNotExist):
			return fmt.Errorf("inspect managed homes directory: %w", err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
	}
}

func (s *Store) RemoveHome(name string) error {
	path := s.Home(name)
	root, err := filepath.Abs(s.Paths.Homes)
	if err != nil {
		return fmt.Errorf("resolve managed home directory: %w", err)
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve isolated home: %w", err)
	}
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == "." || rel == ".." || len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator) {
		return fmt.Errorf("refusing to delete path outside managed homes: %s", path)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("resolve managed homes: %w", err)
	}
	if resolvedRoot != root {
		return fmt.Errorf("refusing to delete through a symlinked managed homes directory")
	}
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect isolated home: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("refusing to delete non-directory isolated home: %s", target)
	}
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("delete isolated home: %w", err)
	}
	return nil
}
