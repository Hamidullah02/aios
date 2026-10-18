package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Checkpoint is the persisted state of a run
// (REQ-RT-DAT-001 / REQ-RT-DAT-002 / TRD §7.1).
//
// The schema is the doc-set's six fields: task_id, plan,
// task_index, context, budget, saved_at. The `context` is a
// free-form JSON object that the worker can populate with
// whatever it needs to resume.
type Checkpoint struct {
	TaskID    string          `json:"task_id"`
	Plan      json.RawMessage `json:"plan"`
	TaskIndex int             `json:"task_index"`
	Context   json.RawMessage `json:"context"`
	Budget    Budget          `json:"budget"`
	SavedAt   string          `json:"saved_at"`
}

// ResumeHandler saves and loads checkpoints
// (REQ-RT-RSM-001 to 003). The default directory is
// `var/checkpoints/`, configurable via the constructor.
type ResumeHandler struct {
	dir string
}

// NewResumeHandler returns a handler that writes to dir. The
// directory is created on the first Save.
func NewResumeHandler(dir string) *ResumeHandler {
	if dir == "" {
		dir = "var/checkpoints"
	}
	return &ResumeHandler{dir: dir}
}

// Save writes the checkpoint to disk. The filename is the
// task id with `.json` appended (REQ-RT-DAT-002). The parent
// directory is created on demand.
func (h *ResumeHandler) Save(cp *Checkpoint) error {
	if cp.TaskID == "" {
		return fmt.Errorf("runtime: checkpoint: task_id is required")
	}
	if err := os.MkdirAll(h.dir, 0o755); err != nil {
		return fmt.Errorf("runtime: checkpoint mkdir: %w", err)
	}
	cp.SavedAt = time.Now().UTC().Format(time.RFC3339Nano)
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("runtime: checkpoint marshal: %w", err)
	}
	path := filepath.Join(h.dir, cp.TaskID+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("runtime: checkpoint write: %w", err)
	}
	return nil
}

// Load reads the checkpoint for taskID. Returns
// ErrCheckpointNotFound when no file exists.
func (h *ResumeHandler) Load(taskID string) (*Checkpoint, error) {
	if taskID == "" {
		return nil, fmt.Errorf("runtime: checkpoint: task_id is required")
	}
	path := filepath.Join(h.dir, taskID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrCheckpointNotFound
		}
		return nil, fmt.Errorf("runtime: checkpoint read: %w", err)
	}
	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("runtime: checkpoint parse: %w", err)
	}
	return &cp, nil
}

// Delete removes the checkpoint for taskID. It is a no-op
// when the file does not exist.
func (h *ResumeHandler) Delete(taskID string) error {
	path := filepath.Join(h.dir, taskID+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ErrCheckpointNotFound is the sentinel for "no checkpoint
// for this task id".
var ErrCheckpointNotFound = fmt.Errorf("runtime: checkpoint not found")
