package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hocicopollo02/sandbox/internal/metadata"
)

type ContainerClient interface {
	Available() error
	Create(ctx context.Context, name, image, home string) error
	Enter(ctx context.Context, name string) error
	Exec(ctx context.Context, name string, command []string) error
	Stop(ctx context.Context, name string) error
	Delete(ctx context.Context, name string) error
}

type Inspector interface {
	Available() error
	Status(ctx context.Context, name string) (Status, error)
	Statuses(ctx context.Context, names []string) (map[string]Status, error)
}

type Manager struct {
	Store     *metadata.Store
	Container ContainerClient
	Inspector Inspector
}

func NewManager(store *metadata.Store, container ContainerClient, inspector Inspector) *Manager {
	return &Manager{Store: store, Container: container, Inspector: inspector}
}

func (m *Manager) Preflight() error {
	if err := m.Container.Available(); err != nil {
		return err
	}
	if err := m.Inspector.Available(); err != nil {
		return err
	}
	return nil
}

func (m *Manager) Create(ctx context.Context, options CreateOptions) (bool, error) {
	if err := options.Validate(); err != nil {
		return false, err
	}
	name, _ := ValidateName(options.Name)
	options.Name = name

	if _, err := m.Store.Get(name); err == nil {
		if options.IfNotExists {
			status, statusErr := m.Inspector.Status(ctx, name)
			if statusErr != nil || status != Missing {
				return false, nil
			}
			if err := m.Store.Delete(name); err != nil {
				return false, err
			}
		} else {
			return false, fmt.Errorf("sandbox %q already exists", name)
		}
	} else if !errors.Is(err, metadata.ErrNotFound) {
		return false, err
	}
	if err := m.Preflight(); err != nil {
		return false, err
	}
	status, err := m.Inspector.Status(ctx, name)
	if err != nil {
		return false, err
	}
	if status != Missing {
		return false, fmt.Errorf("sandbox %q already exists outside sandbox metadata", name)
	}
	home := ""
	if options.HomeMode == IsolatedHome {
		if err := m.Store.ValidateHomesRoot(); err != nil {
			return false, err
		}
		home = m.Store.Home(name)
		if _, err := os.Stat(home); err == nil {
			return false, fmt.Errorf("isolated home already exists: %s", home)
		} else if !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("check isolated home: %w", err)
		}
	}

	// Reserve the name atomically before creating any resource; a losing
	// concurrent create fails here without touching the winner's home.
	record := Record{
		Name:         name,
		Distribution: options.Distribution.ID,
		Image:        options.Distribution.Image,
		Persistence:  options.Persistence,
		HomeMode:     options.HomeMode,
		CreatedAt:    time.Now().In(time.Local),
	}
	if err := m.Store.SaveExclusive(record); err != nil {
		if options.IfNotExists && errors.Is(err, os.ErrExist) {
			return false, nil
		}
		return false, err
	}
	discardReservation := func() error { return m.Store.Delete(name) }

	homeCreated := false
	if options.HomeMode == IsolatedHome {
		if err := os.MkdirAll(home, 0700); err != nil {
			_ = discardReservation()
			return false, fmt.Errorf("create isolated home: %w", err)
		}
		homeCreated = true
	}
	cleanup := func() error {
		err := discardReservation()
		if homeCreated {
			err = errors.Join(err, m.Store.RemoveHome(name))
		}
		return err
	}

	if err := m.Container.Create(ctx, name, options.Distribution.Image, home); err != nil {
		return false, errors.Join(err, cleanup())
	}

	if options.Persistence == Persistent {
		record.HomePath = home
		if err := m.Store.Save(record); err != nil {
			cleanupErr := errors.Join(m.Container.Delete(ctx, name), cleanup())
			return false, errors.Join(fmt.Errorf("save sandbox metadata: %w", err), cleanupErr)
		}
	}

	if !options.AutoEnter {
		return false, nil
	}
	if err := m.Container.Enter(ctx, name); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cleanupErr := errors.Join(m.Container.Delete(cleanupCtx, name), discardReservation())
		if homeCreated {
			cleanupErr = errors.Join(cleanupErr, m.Store.RemoveHome(name))
		}
		return false, errors.Join(err, cleanupErr)
	}

	if options.Persistence == Disposable {
		deleteErr := m.Container.Delete(ctx, name)
		var homeErr error
		if homeCreated {
			homeErr = m.Store.RemoveHome(name)
		}
		metaErr := m.Store.Delete(name)
		if deleteErr != nil || homeErr != nil || metaErr != nil {
			return true, errors.Join(deleteErr, homeErr, metaErr)
		}
		return true, nil
	}
	return false, nil
}

func (m *Manager) Enter(ctx context.Context, name string) error {
	name, err := ValidateName(name)
	if err != nil {
		return err
	}
	if _, err := m.Store.Get(name); errors.Is(err, metadata.ErrNotFound) {
		return fmt.Errorf("sandbox %q does not exist", name)
	} else if err != nil {
		return err
	}
	status, err := m.Inspector.Status(ctx, name)
	if err != nil {
		return err
	}
	if status == Missing {
		return fmt.Errorf("sandbox %q does not exist in the container runtime", name)
	}
	return m.Container.Enter(ctx, name)
}

// Exec runs a command inside a sandbox without a TTY, reusing the same
// existence and status checks as Enter.
func (m *Manager) Exec(ctx context.Context, name string, command []string) error {
	name, err := ValidateName(name)
	if err != nil {
		return err
	}
	if _, err := m.Store.Get(name); errors.Is(err, metadata.ErrNotFound) {
		return fmt.Errorf("sandbox %q does not exist", name)
	} else if err != nil {
		return err
	}
	status, err := m.Inspector.Status(ctx, name)
	if err != nil {
		return err
	}
	if status == Missing {
		return fmt.Errorf("sandbox %q does not exist in the container runtime", name)
	}
	if status == Unknown {
		return fmt.Errorf("could not determine the status of sandbox %q", name)
	}
	return m.Container.Exec(ctx, name, command)
}

func (m *Manager) List(ctx context.Context) ([]ListEntry, error) {
	records, err := m.Store.List()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(records))
	for _, record := range records {
		names = append(names, record.Name)
	}
	statuses, err := m.Inspector.Statuses(ctx, names)
	if err != nil {
		return nil, err
	}
	entries := make([]ListEntry, 0, len(records))
	for _, record := range records {
		entries = append(entries, ListEntry{
			Name:         record.Name,
			Distribution: record.Distribution,
			Persistence:  record.Persistence,
			HomeMode:     record.HomeMode,
			Status:       statuses[record.Name],
		})
	}
	return entries, nil
}

func (m *Manager) Stop(ctx context.Context, name string) error {
	name, err := ValidateName(name)
	if err != nil {
		return err
	}
	if _, err := m.Store.Get(name); errors.Is(err, metadata.ErrNotFound) {
		return fmt.Errorf("sandbox %q does not exist", name)
	} else if err != nil {
		return err
	}
	status, err := m.Inspector.Status(ctx, name)
	if err != nil {
		return err
	}
	if status == Missing {
		return fmt.Errorf("sandbox %q does not exist in the container runtime", name)
	}
	if status == Stopped {
		return nil
	}
	return m.Container.Stop(ctx, name)
}

func (m *Manager) Info(ctx context.Context, name string) (Info, error) {
	name, err := ValidateName(name)
	if err != nil {
		return Info{}, err
	}
	record, err := m.Store.Get(name)
	if errors.Is(err, metadata.ErrNotFound) {
		return Info{}, fmt.Errorf("sandbox %q does not exist", name)
	}
	if err != nil {
		return Info{}, err
	}
	status, err := m.Inspector.Status(ctx, name)
	if err != nil {
		return Info{}, err
	}
	return Info{Record: record, Status: status}, nil
}

// Lookup returns stored metadata without querying the container runtime; the
// Status field is intentionally empty.
func (m *Manager) Lookup(_ context.Context, name string) (Info, error) {
	name, err := ValidateName(name)
	if err != nil {
		return Info{}, err
	}
	record, err := m.Store.Get(name)
	if errors.Is(err, metadata.ErrNotFound) {
		return Info{}, fmt.Errorf("sandbox %q does not exist: %w", name, metadata.ErrNotFound)
	}
	if err != nil {
		return Info{}, err
	}
	return Info{Record: record}, nil
}

func (m *Manager) Delete(ctx context.Context, name string, options DeleteOptions) error {
	name, err := ValidateName(name)
	if err != nil {
		return err
	}
	record, err := m.Store.Get(name)
	if errors.Is(err, metadata.ErrNotFound) {
		if options.IfExists {
			return nil
		}
		return fmt.Errorf("sandbox %q does not exist", name)
	}
	if err != nil {
		return err
	}
	if options.DeleteHome && record.HomeMode == IsolatedHome {
		expected, err := filepath.Abs(filepath.Clean(m.Store.Home(name)))
		if err != nil {
			return fmt.Errorf("resolve managed isolated home: %w", err)
		}
		actual, err := filepath.Abs(filepath.Clean(record.HomePath))
		if err != nil || actual != expected {
			return fmt.Errorf("refusing to delete isolated home outside managed directory")
		}
	}
	status, err := m.Inspector.Status(ctx, name)
	if err != nil {
		return err
	}
	if status == Unknown {
		return fmt.Errorf("could not determine the status of sandbox %q", name)
	}
	if status != Missing {
		if err := m.Container.Delete(ctx, name); err != nil {
			return fmt.Errorf("delete container for sandbox %q: %w", name, err)
		}
	}
	if options.DeleteHome && record.HomeMode == IsolatedHome {
		if err := m.Store.RemoveHome(name); err != nil {
			return err
		}
	}
	return m.Store.Delete(name)
}

func (m *Manager) ManagedHome(name string) string {
	return filepath.Clean(m.Store.Home(name))
}

func SubIDConfigured(path, username, uid string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.SplitN(strings.TrimSpace(line), ":", 3)
		if len(fields) == 3 && (fields[0] == username || fields[0] == uid) {
			return true
		}
	}
	return false
}
