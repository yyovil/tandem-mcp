package prompts

import _ "embed"

var (
	//go:embed instructions.md
	Instructions string

	//go:embed terminal.md
	Terminal string
)
