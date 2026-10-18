package git

// GitStatus is the working-tree state (REQ-GIT-DAT-001).
type GitStatus struct {
	Branch    string   `json:"branch"`
	Staged    []string `json:"staged"`
	Modified  []string `json:"modified"`
	Untracked []string `json:"untracked"`
}

// GitCommit is a parsed commit row (REQ-GIT-DAT-002).
type GitCommit struct {
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Subject string `json:"subject"`
}
