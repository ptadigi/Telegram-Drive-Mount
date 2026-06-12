package vfs

import (
	"context"
	"errors"
	"sync"
	"time"

	"telegram-drive-agent/internal/drive"
)

// Status describes the current mount state.
type Status struct {
	Available   bool   `json:"available"`
	Mounted     bool   `json:"mounted"`
	MountPoint  string `json:"mount_point,omitempty"`
	DriveLetter string `json:"drive_letter,omitempty"`
	Backend     string `json:"backend"`
	Error       string `json:"error,omitempty"`
	StartedAt   int64  `json:"started_at,omitempty"`
}

// Mounter is implemented by platform-specific mount engines (cgofuse/WinFsp/FUSE).
type Mounter interface {
	Backend() string
	Mount(ctx context.Context, mountPoint string) error
	Unmount() error
	IsMounted() bool
	MountPoint() string
}

// Manager orchestrates a Mounter with thread-safe state.
type Manager struct {
	mu       sync.Mutex
	mounter  Mounter
	backend  Backend
	dataDir  string
	status   Status
	cancel   context.CancelFunc
	mounting bool
}

type syncWatcherEntry struct {
	cancel    context.CancelFunc
	updatedAt int64
}

// NewManager builds a manager for the active build (fuse-enabled or stub).
func NewManager(svc *drive.Service, dataDir string) *Manager {
	backend := newLocalBackend(svc)
	m := &Manager{backend: backend, dataDir: dataDir}
	m.mounter = newPlatformMounter(backend, dataDir)
	if m.mounter == nil {
		m.status = Status{Available: false, Backend: "none"}
	} else {
		m.status = Status{Available: true, Backend: m.mounter.Backend()}
	}
	return m
}

// NewManagerWithBackend lets callers (e.g. td-agent --remote) supply a
// custom Backend implementation that talks to a remote VPS over HTTPS.
func NewManagerWithBackend(backend Backend, dataDir string) *Manager {
	m := &Manager{backend: backend, dataDir: dataDir}
	m.mounter = newPlatformMounter(backend, dataDir)
	if m.mounter == nil {
		m.status = Status{Available: false, Backend: "none"}
	} else {
		m.status = Status{Available: true, Backend: m.mounter.Backend()}
	}
	return m
}

// Status returns a snapshot.
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.mounter != nil {
		mounted := m.mounter.IsMounted()
		m.status.Mounted = mounted
		if mounted {
			if mp := m.mounter.MountPoint(); mp != "" {
				m.status.MountPoint = mp
			}
			m.status.Error = ""
		}
	}
	return m.status
}

// Mount kicks off a mount in the background and returns once mounter reports ready.
func (m *Manager) Mount(ctx context.Context, mountPoint string) (Status, error) {
	m.mu.Lock()
	if m.mounter == nil {
		m.mu.Unlock()
		return m.status, errors.New("mount engine không có sẵn trong bản build này")
	}
	if m.mounting {
		current := m.status
		m.mu.Unlock()
		return current, errors.New("đang trong quá trình mount, vui lòng chờ")
	}
	if m.mounter.IsMounted() {
		current := m.status
		m.mu.Unlock()
		return current, errors.New("ổ ảo đã được mount")
	}
	resolved := mountPoint
	if resolved == "" {
		resolved = defaultMountPoint()
	}
	runCtx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.mounting = true
	m.status.MountPoint = resolved
	m.status.Mounted = false
	m.status.Error = ""
	m.status.StartedAt = time.Now().Unix()
	m.mu.Unlock()

	resultCh := make(chan error, 1)
	go func() {
		resultCh <- m.mounter.Mount(runCtx, resolved)
	}()

	deadline := time.After(8 * time.Second)
	for {
		if m.mounter.IsMounted() {
			m.mu.Lock()
			m.status.Mounted = true
			m.status.MountPoint = m.mounter.MountPoint()
			m.mounting = false
			snap := m.status
			m.mu.Unlock()
			return snap, nil
		}
		select {
		case err := <-resultCh:
			m.mu.Lock()
			m.mounting = false
			if err != nil {
				m.status.Error = err.Error()
				m.status.Mounted = false
			} else {
				m.status.Mounted = m.mounter.IsMounted()
				if m.status.Mounted {
					m.status.MountPoint = m.mounter.MountPoint()
				}
			}
			snap := m.status
			m.mu.Unlock()
			return snap, err
		case <-deadline:
			m.mu.Lock()
			m.mounting = false
			snap := m.status
			m.mu.Unlock()
			return snap, nil
		case <-ctx.Done():
			cancel()
			m.mu.Lock()
			m.mounting = false
			snap := m.status
			m.mu.Unlock()
			return snap, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// Unmount stops the mounter.
func (m *Manager) Unmount() (Status, error) {
	m.mu.Lock()
	cancel := m.cancel
	m.cancel = nil
	mounter := m.mounter
	m.mu.Unlock()
	if mounter == nil {
		return m.Status(), errors.New("mount engine không có sẵn")
	}
	if cancel != nil {
		cancel()
	}
	if err := mounter.Unmount(); err != nil {
		m.mu.Lock()
		m.status.Error = err.Error()
		snap := m.status
		m.mu.Unlock()
		return snap, err
	}
	m.mu.Lock()
	m.status.Mounted = false
	m.status.Error = ""
	snap := m.status
	m.mu.Unlock()
	return snap, nil
}

// Shutdown stops the mounter on agent exit.
func (m *Manager) Shutdown() {
	if m == nil || m.mounter == nil {
		return
	}
	if m.mounter.IsMounted() {
		_, _ = m.Unmount()
	}
}

// SwitchBackend swaps the active backend at runtime and (re)mounts at
// mountPoint. This lets the desktop app change between local and remote modes
// without restarting the agent: after pairing/unpairing we rebuild the mounter
// on the new backend so the virtual drive immediately reflects the right
// source. It unmounts any current mount first. A no-op safe guard: if the new
// mounter can't be built, the previous state is left untouched.
func (m *Manager) SwitchBackend(backend Backend, mountPoint string) (Status, error) {
	if m == nil {
		return Status{}, errors.New("mount manager chưa khởi tạo")
	}
	// Tear down the current mount (best-effort) before swapping.
	if m.mounter != nil && m.mounter.IsMounted() {
		_, _ = m.Unmount()
	}

	mounter := newPlatformMounter(backend, m.dataDir)
	m.mu.Lock()
	m.backend = backend
	m.mounter = mounter
	if mounter == nil {
		m.status = Status{Available: false, Backend: "none"}
		m.mu.Unlock()
		return m.status, errors.New("mount engine không có sẵn trong bản build này")
	}
	m.status = Status{Available: true, Backend: mounter.Backend()}
	m.mu.Unlock()

	resolved := mountPoint
	if resolved == "" {
		resolved = defaultMountPoint()
	}
	return m.Mount(context.Background(), resolved)
}
