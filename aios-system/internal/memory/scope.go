package memory

import (
	"fmt"
	"sync"
)

// Memory is the top-level memory container.
//
// The container owns the per-scope (Store, Index)
// pairs and the PinSet. The container is the object
// callers use; the per-scope objects are the
// implementation detail.
//
// The three scopes are:
//
//   - session: in-memory only. The session store and
//     index are empty by default and are reset on
//     Close. The session dies when the session ends.
//   - project: persistent. The project store and
//     index live at <path>/project/ and are
//     git-trackable.
//   - user: persistent. The user store and index live
//     at <home>/.aios/memory/ and are optionally
//     encrypted at rest through AES-GCM.
type Memory struct {
	mu sync.RWMutex

	session *ScopeView
	project *ScopeView
	user    *ScopeView
	pinSet  *PinSet
}

// MemoryConfig controls Memory construction.
type MemoryConfig struct {
	// DataDir is the project-level data directory.
	// Defaults to $AIOS_DATA_DIR or "./var/". The
	// project scope lives at <DataDir>/memory/.
	DataDir string
	// Home is the user-level home. Defaults to
	// $AIOS_HOME or $HOME/.aios/. The user scope
	// lives at <Home>/memory/.
	Home string
	// UserEncryptor is the optional encryptor for the
	// user scope. When nil, the user scope is
	// plaintext (suitable for development).
	// Production deployments that hold secrets should
	// always provide an encryptor.
	UserEncryptor Encryptor
}

// NewMemory constructs a Memory. The constructor
// loads the project and user scopes from disk when
// they exist; missing directories are created. The
// session scope is created in memory.
//
// Per REQ-MEM-SCO-001 the memory supports three
// scopes. Per REQ-MEM-SCO-002 the session scope is
// in-memory only. Per REQ-MEM-SCO-003 the project
// scope is at <DataDir>/memory/. Per REQ-MEM-SCO-004
// the user scope is at <Home>/memory/ and is
// optionally encrypted at rest.
func NewMemory(cfg MemoryConfig) (*Memory, error) {
	dataDir := cfg.DataDir
	if dataDir == "" {
		dataDir = defaultDataDir()
	}
	home := cfg.Home
	if home == "" {
		home = defaultHome()
	}

	project, err := NewScopeView(ScopeViewConfig{
		Scope:     ScopeProject,
		BasePath:  fmt.Sprintf("%s/memory", dataDir),
		Encryptor: nil,
	})
	if err != nil {
		return nil, &MemoryError{Op: "init", Reason: fmt.Sprintf("project scope: %s", err.Error())}
	}
	user, err := NewScopeView(ScopeViewConfig{
		Scope:     ScopeUser,
		BasePath:  fmt.Sprintf("%s/memory", home),
		Encryptor: cfg.UserEncryptor,
	})
	if err != nil {
		return nil, &MemoryError{Op: "init", Reason: fmt.Sprintf("user scope: %s", err.Error())}
	}
	session, err := NewScopeView(ScopeViewConfig{
		Scope:    ScopeSession,
		BasePath: "",
	})
	if err != nil {
		return nil, &MemoryError{Op: "init", Reason: fmt.Sprintf("session scope: %s", err.Error())}
	}

	pinSet, err := NewPinSet(PinSetConfig{
		Path: fmt.Sprintf("%s/memory", dataDir),
	})
	if err != nil {
		return nil, &MemoryError{Op: "init", Reason: fmt.Sprintf("pin set: %s", err.Error())}
	}

	return &Memory{
		session: session,
		project: project,
		user:    user,
		pinSet:  pinSet,
	}, nil
}

// Remember stores an artifact. The raw payload is
// put in the store; the artifact is added to the
// index. The two operations are atomic through a
// mutex.
//
// The function returns the content hash. The hash
// is the canonical id of the artifact; the caller
// can use it for queries and pins.
//
// REQ-MEM-STO-003 + REQ-MEM-IDX-003.
func (m *Memory) Remember(a Artifact) (string, error) {
	if err := a.Validate(); err != nil {
		return "", err
	}
	view, err := m.scopeView(a.Scope)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := view.putLocked(a); err != nil {
		return "", err
	}
	return a.ContentHash, nil
}

// Get returns the artifact for the content hash. The
// function searches all three scopes in order:
// session, project, user. The first match wins.
//
// The function returns ErrNotFound when the hash is
// not in any scope.
func (m *Memory) Get(hash string) (Artifact, error) {
	if hash == "" {
		return Artifact{}, &MemoryError{Op: "get", Reason: "hash is required"}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, view := range []*ScopeView{m.session, m.project, m.user} {
		entry, err := view.index.Get(hash)
		if err != nil {
			continue
		}
		payload, err := view.store.Get(hash)
		if err != nil {
			return Artifact{}, &MemoryError{Op: "get", Reason: err.Error()}
		}
		return entryToArtifact(entry, payload), nil
	}
	return Artifact{}, ErrNotFound
}

// Has reports whether the artifact is in any scope.
func (m *Memory) Has(hash string) bool {
	if hash == "" {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, view := range []*ScopeView{m.session, m.project, m.user} {
		if view.index.Has(hash) {
			return true
		}
	}
	return false
}

// Query runs a query against all three scopes and
// returns the matching artifacts. The function is
// the read-side counterpart to Remember.
//
// The function returns the merged, sorted result.
// The limit, when set, is applied to the merged
// result so a small limit still draws from every
// scope.
func (m *Memory) Query(q Query) ([]Artifact, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	all := m.session.index.All()
	all = append(all, m.project.index.All()...)
	all = append(all, m.user.index.All()...)
	matches, err := q.RunIndex(all)
	if err != nil {
		return nil, err
	}
	out := make([]Artifact, 0, len(matches))
	for _, entry := range matches {
		view, err := m.scopeViewLocked(entry.Scope)
		if err != nil {
			return nil, err
		}
		payload, err := view.store.Get(entry.ContentHash)
		if err != nil {
			// Index entry without payload is a
			// corrupt state. Skip and continue.
			continue
		}
		out = append(out, entryToArtifact(entry, payload))
	}
	return out, nil
}

// Session returns the session scope view. The
// function is the scope getter for the resume and
// cancel flows.
func (m *Memory) Session() *ScopeView { return m.session }

// Project returns the project scope view.
func (m *Memory) Project() *ScopeView { return m.project }

// User returns the user scope view.
func (m *Memory) User() *ScopeView { return m.user }

// PinSet returns the pin set. The function is the
// pin getter for the developer CLI and the retrieval
// algorithm.
func (m *Memory) PinSet() *PinSet { return m.pinSet }

// scopeView returns the scope view for the given
// scope. The function is the dispatch the remember
// flow uses.
func (m *Memory) scopeView(s Scope) (*ScopeView, error) {
	switch s {
	case ScopeSession:
		return m.session, nil
	case ScopeProject:
		return m.project, nil
	case ScopeUser:
		return m.user, nil
	}
	return nil, &MemoryError{Op: "scope", Reason: fmt.Sprintf("unknown scope %q", s)}
}

// scopeViewLocked returns the scope view for the
// given scope. The function is the dispatch the
// query flow uses; it assumes the caller holds the
// memory's mutex.
func (m *Memory) scopeViewLocked(s Scope) (*ScopeView, error) {
	switch s {
	case ScopeSession:
		return m.session, nil
	case ScopeProject:
		return m.project, nil
	case ScopeUser:
		return m.user, nil
	}
	return nil, &MemoryError{Op: "scope", Reason: fmt.Sprintf("unknown scope %q", s)}
}

// ResetSession clears the session scope. The
// function is the lifecycle hook the session-end
// flow calls; the session store and index are
// dropped and rebuilt empty.
func (m *Memory) ResetSession() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, err := NewScopeView(ScopeViewConfig{
		Scope:    ScopeSession,
		BasePath: "",
	})
	if err != nil {
		return &MemoryError{Op: "reset session", Reason: err.Error()}
	}
	m.session = s
	return nil
}

// ScopeView is the per-scope (Store, Index) pair.
//
// The session scope view is built in memory; the
// project and user scope views are built on disk.
// The two views share the same API: Put, Get, Index.
type ScopeView struct {
	scope Scope
	store *Store
	index *Index
}

// ScopeViewConfig controls ScopeView construction.
type ScopeViewConfig struct {
	// Scope is the scope this view represents.
	Scope Scope
	// BasePath is the directory under which the
	// store and index live. Empty for the session
	// scope; the view is built in memory.
	BasePath string
	// Encryptor is the optional encryptor for the
	// store. Used by the user scope.
	Encryptor Encryptor
}

// NewScopeView constructs a ScopeView. The session
// scope is built in memory when BasePath is empty;
// the project and user scopes are built on disk.
func NewScopeView(cfg ScopeViewConfig) (*ScopeView, error) {
	view := &ScopeView{scope: cfg.Scope}
	if cfg.Scope == ScopeSession {
		// Session scope: in-memory only.
		view.store = newInMemoryStore()
		idx, err := NewIndex(IndexConfig{Path: ""})
		if err != nil {
			return nil, err
		}
		view.index = idx
		return view, nil
	}
	if cfg.BasePath == "" {
		return nil, &MemoryError{Op: "init", Reason: "base path is required for persistent scopes"}
	}
	store, err := NewStore(StoreConfig{
		Path:      cfg.BasePath + "/raw",
		Encryptor: cfg.Encryptor,
	})
	if err != nil {
		return nil, err
	}
	idx, err := NewIndex(IndexConfig{Path: cfg.BasePath + "/index"})
	if err != nil {
		return nil, err
	}
	view.store = store
	view.index = idx
	return view, nil
}

// Scope returns the view's scope.
func (v *ScopeView) Scope() Scope { return v.scope }

// Store returns the view's store. The function is
// the low-level getter tests use.
func (v *ScopeView) Store() *Store { return v.store }

// Index returns the view's index. The function is
// the low-level getter the retrieval algorithm uses.
func (v *ScopeView) Index() *Index { return v.index }

// putLocked stores the artifact in the view. The
// function is called with the memory's mutex held.
func (v *ScopeView) putLocked(a Artifact) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if a.Scope != v.scope {
		return &MemoryError{Op: "put", Reason: fmt.Sprintf("scope mismatch: view is %q, artifact is %q", v.scope, a.Scope)}
	}
	if _, err := v.store.Put(a.Content); err != nil {
		return err
	}
	if err := v.index.Add(a); err != nil {
		return err
	}
	return nil
}

// entryToArtifact is the join helper that rebuilds
// an Artifact from an index entry and the raw
// payload. The function is the inverse of NewArtifact.
func entryToArtifact(e IndexEntry, payload []byte) Artifact {
	return Artifact{
		Kind:        e.Kind,
		Scope:       e.Scope,
		SessionID:   e.SessionID,
		TaskID:      e.TaskID,
		Timestamp:   e.Timestamp,
		Tags:        append([]string(nil), e.Tags...),
		Content:     payload,
		ContentHash: e.ContentHash,
	}
}

// defaultDataDir returns the project data directory.
// The path is $AIOS_DATA_DIR when set, or "./var/".
func defaultDataDir() string {
	if dir := getenv("AIOS_DATA_DIR"); dir != "" {
		return dir
	}
	return "./var"
}

// defaultHome returns the user home. The path is
// $AIOS_HOME when set, or $HOME/.aios/ (or
// %USERPROFILE%/.aios/ on Windows).
func defaultHome() string {
	if home := getenv("AIOS_HOME"); home != "" {
		return home
	}
	if home := getenv("HOME"); home != "" {
		return home + "/.aios"
	}
	if home := getenv("USERPROFILE"); home != "" {
		return home + "/.aios"
	}
	return "./.aios"
}
