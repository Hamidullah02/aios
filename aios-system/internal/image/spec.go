package image

// Describe implements tools.Spec for ReadImageTool.
func (ReadImageTool) Describe() (string, string) {
	return "Read an image file from the workspace and report its dimensions and format. " +
			"Args: path (required, inside the workspace).",
		`{"type":"object","required":["path"],"properties":{"path":{"type":"string"}}}`
}

// Risk implements tools.RiskTier for ReadImageTool.
func (ReadImageTool) Risk() int { return 0 }
