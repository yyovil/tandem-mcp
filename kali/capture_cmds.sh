#!/bin/bash

# Capture commands from the 'agent' tmux window
# Requires: tmux, awk, sed, freeze
# Uses prompt marker: KALI[path]# (set via Dockerfile)
# Usage: Source this script in the 'screenshotter' window, then call showLastNExecution <n>

AGENT_WINDOW="agent"
PROMPT_MARKER="KALI["

# Capture last n command executions and generate an image
# Parameters:
#   $1 - n (int): number of commands to capture (whole numbers: 0, 1, 2, ...)
# Output:
#   PNG image saved to /tmp/last_<n>_cmds.png
showLastNExecution() {
    local n="$1"

    # Validate input
    if ! [[ "$n" =~ ^[0-9]+$ ]]; then
        echo "Error: n must be a non-negative integer" >&2
        return 1
    fi

    if [[ "$n" -eq 0 ]]; then
        echo "Nothing to capture (n=0)" >&2
        return 0
    fi

    # Auto-detect current tmux session
    local session
    session="$(tmux display-message -p '#{session_name}' 2>/dev/null)"
    if [[ -z "$session" ]]; then
        echo "Error: Not running inside a tmux session" >&2
        return 1
    fi

    local target="${session}:${AGENT_WINDOW}"
    local output_file="/tmp/last_${n}_cmds.png"

    # Capture pane, extract last n commands using KALI[path]# prompt, strip trailing whitespace, generate image
    tmux capture-pane -pt "$target" -S - | \
    awk -v N="$n" '
        BEGIN { cmd_count=0; in_cmd=0; buffer=""; has_cmd=0 }
        # Detect prompt line matching KALI[...]# pattern
        /^KALI\[.*\]#/ {
            # Store previous buffer if it had an actual command (not just bare prompt)
            if (in_cmd && has_cmd) {
                cmds[cmd_count] = buffer
                cmd_count++
            }
            in_cmd = 1
            buffer = $0 "\n"
            # Check if there is a command after the prompt (line continues after ]# )
            has_cmd = (match($0, /\]# ./) > 0)
            next
        }
        in_cmd { buffer = buffer $0 "\n" }
        END {
            # Store the last command buffer if it had an actual command
            if (in_cmd && has_cmd) {
                cmds[cmd_count] = buffer
                cmd_count++
            }
            start = cmd_count - N
            if (start < 0) start = 0
            for (i = start; i < cmd_count; i++) {
                printf "%s", cmds[i]
            }
        }
    ' | \
    sed -e :a -e '/^\s*$/{ $d; N; ba; }' | \
    freeze -l bash -o "$output_file"

    if [[ -f "$output_file" ]]; then
        echo "$output_file"
        return 0
    else
        echo "Error: Failed to generate image" >&2
        return 1
    fi
}

# Run if script is executed directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    n="${1:-1}"
    showLastNExecution "$n"
fi
                   