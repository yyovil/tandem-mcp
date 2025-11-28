package prompts

import _ "embed"

var (
	//go:embed instructions.md
	Instructions string

	//go:embed terminal.md
	Terminal string

	//go:embed show_last_n_execution.md
	ShowLastNExecution string
)
