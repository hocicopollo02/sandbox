package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/hocicopollo02/sandbox/internal/model"
	"golang.org/x/sys/unix"
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
	mu    sync.Mutex
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
	s.mu.Lock()
	defer s.mu.Unlock()
	unlock, err := s.lock()
	if err != nil {
		return err
	}
	defer unlock()
	return s.save(record)
}

func (s *Store) save(record model.Record) error {
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

// SaveExclusive atomically reserves the sandbox name: it fails if a record
// already exists, so a losing concurrent create cannot claim the same name.
func (s *Store) SaveExclusive(record model.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	unlock, err := s.lock()
	if err != nil {
		return err
	}
	defer unlock()
	return s.saveExclusive(record)
}

func (s *Store) saveExclusive(record model.Record) error {
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
	file, err := os.OpenFile(s.file(record.Name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("sandbox %q already exists: %w", record.Name, os.ErrExist)
	}
	if err != nil {
		return fmt.Errorf("reserve metadata for %q: %w", record.Name, err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write metadata: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close metadata: %w", err)
	}
	return nil
}

func (s *Store) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	unlock, err := s.lock()
	if err != nil {
		return err
	}
	defer unlock()
	return s.delete(name)
}

func (s *Store) delete(name string) error {
	if err := os.Remove(s.file(name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete metadata for %q: %w", name, err)
	}
	return nil
}

func (s *Store) DeleteIfMatch(expected model.Record, cleanup func() error) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	unlock, err := s.lock()
	if err != nil {
		return false, err
	}
	defer unlock()
	record, err := s.Get(expected.Name)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !reflect.DeepEqual(record, expected) {
		return false, nil
	}
	if err := s.delete(expected.Name); err != nil {
		return false, err
	}
	if cleanup != nil {
		if err := cleanup(); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (s *Store) lock() (func(), error) {
	if err := os.MkdirAll(s.Paths.Metadata, 0700); err != nil {
		return nil, fmt.Errorf("create metadata directory: %w", err)
	}
	file, err := os.OpenFile(filepath.Join(s.Paths.Metadata, ".sandbox.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open metadata lock: %w", err)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("lock metadata: %w", err)
	}
	return func() {
		_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
		_ = file.Close()
	}, nil
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
		// Read-only content such as a Go module cache blocks unlink from the host;
		// grant the owner write permission and retry once.
		if werr := grantWritePermission(target); werr != nil {
			return fmt.Errorf("delete isolated home: %w; could not make the content writable: %w", err, werr)
		}
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("delete isolated home after granting write permission: %w", err)
		}
	}
	return nil
}

func grantWritePermission(root string) error {
	var walkErr error
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, werr error) error {
		if werr != nil {
			walkErr = werr
			return werr
		}
		info, err := entry.Info()
		if err != nil {
			walkErr = err
			return err
		}
		mode := info.Mode().Perm() | 0200
		if info.IsDir() {
			mode |= 0100
		}
		return os.Chmod(path, mode)
	})
	return errors.Join(walkErr, err)
}
