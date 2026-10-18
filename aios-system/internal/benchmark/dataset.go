package benchmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// WedgeTask is one task in the wedge dataset
// (TRD §7.2).
type WedgeTask struct {
	ID             string         `json:"id"`
	Wedge          string         `json:"wedge"`
	Goal           string         `json:"goal"`
	ExpectedResult map[string]any `json:"expected_result,omitempty"`
	TimeoutSeconds int            `json:"timeout_seconds,omitempty"`
}

// WedgeDataset is a list of tasks (REQ-BCH-DAT-001).
type WedgeDataset struct {
	Tasks []WedgeTask `json:"tasks"`
}

// Wedge IDs from the SRS (REQ-BCH-DAT-002).
const (
	WedgeFileRead   = "file_read"
	WedgeFileWrite  = "file_write"
	WedgeFileSearch = "file_search"
	WedgeShellRun   = "shell_run"
	WedgeApproval   = "approval"
	WedgeAskUser    = "ask_user"
	WedgeResume     = "resume"
	WedgeCancel     = "cancel"
)

// DefaultDataset is the built-in wedge dataset. It has
// one task per wedge (REQ-BCH-DAT-002) so the loop and
// two-class runners have non-empty inputs by default.
func DefaultDataset() WedgeDataset {
	return WedgeDataset{Tasks: []WedgeTask{
		{ID: "fr-1", Wedge: WedgeFileRead, Goal: "Read file a.txt and return its content.", TimeoutSeconds: 30},
		{ID: "fw-1", Wedge: WedgeFileWrite, Goal: "Write 'hello' to file a.txt.", TimeoutSeconds: 30},
		{ID: "fs-1", Wedge: WedgeFileSearch, Goal: "Find all .go files in the workspace.", TimeoutSeconds: 30},
		{ID: "sh-1", Wedge: WedgeShellRun, Goal: "Run `go version` and return the output.", TimeoutSeconds: 30},
		{ID: "ap-1", Wedge: WedgeApproval, Goal: "Write to a file that requires approval.", TimeoutSeconds: 60},
		{ID: "au-1", Wedge: WedgeAskUser, Goal: "Ask the user a clarifying question.", TimeoutSeconds: 30},
		{ID: "rs-1", Wedge: WedgeResume, Goal: "Resume a paused task.", TimeoutSeconds: 60},
		{ID: "cn-1", Wedge: WedgeCancel, Goal: "Cancel a running task.", TimeoutSeconds: 30},
	}}
}

// LoadDataset reads a JSON file from path. The file
// must have a top-level {"tasks": [...]} structure
// (REQ-BCH-DAT-003). The default is `./benchmark/wedge.json`,
// the doc-set default.
func LoadDataset(path string) (WedgeDataset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return WedgeDataset{}, fmt.Errorf("benchmark: dataset: read %q: %w", path, err)
	}
	var ds WedgeDataset
	if err := json.Unmarshal(data, &ds); err != nil {
		return WedgeDataset{}, fmt.Errorf("benchmark: dataset: parse %q: %w", path, err)
	}
	if err := ds.Validate(); err != nil {
		return WedgeDataset{}, err
	}
	return ds, nil
}

// Save writes the dataset to path. Round-trips with
// LoadDataset.
func (d WedgeDataset) Save(path string) error {
	if err := d.Validate(); err != nil {
		return err
	}
	canon, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("benchmark: dataset: marshal: %w", err)
	}
	return os.WriteFile(path, canon, 0o644)
}

// Validate checks that every task has a non-empty id,
// a non-empty wedge, and a non-empty goal. Returns
// the first error found.
func (d WedgeDataset) Validate() error {
	seen := make(map[string]bool, len(d.Tasks))
	for i, t := range d.Tasks {
		if t.ID == "" {
			return fmt.Errorf("benchmark: dataset: task %d: id is empty", i)
		}
		if seen[t.ID] {
			return fmt.Errorf("benchmark: dataset: duplicate task id %q", t.ID)
		}
		seen[t.ID] = true
		if t.Wedge == "" {
			return fmt.Errorf("benchmark: dataset: task %q: wedge is empty", t.ID)
		}
		if t.Goal == "" {
			return fmt.Errorf("benchmark: dataset: task %q: goal is empty", t.ID)
		}
		if t.TimeoutSeconds < 0 {
			return fmt.Errorf("benchmark: dataset: task %q: timeout_seconds is negative", t.ID)
		}
	}
	return nil
}

// FilterByWedge returns a new dataset with only the
// tasks whose Wedge matches one of the given ids. An
// empty wedge list returns the full dataset.
func (d WedgeDataset) FilterByWedge(wedges ...string) WedgeDataset {
	if len(wedges) == 0 {
		return d
	}
	keep := make(map[string]bool, len(wedges))
	for _, w := range wedges {
		keep[w] = true
	}
	var out []WedgeTask
	for _, t := range d.Tasks {
		if keep[t.Wedge] {
			out = append(out, t)
		}
	}
	return WedgeDataset{Tasks: out}
}

// ErrEmptyDataset is returned when a runner is asked
// to run a dataset with no tasks.
var ErrEmptyDataset = errors.New("benchmark: dataset is empty")
