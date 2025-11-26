#!/bin/bash

# Start a new tmux session named "pentest" with the first window named "agent"
tmux new-session -d -s pentest -n agent

# Create a second window named "screenshotter" in the same session
tmux new-window -t pentest -n screenshotter

# Select the first window (agent) as the default active window
tmux select-window -t pentest:agent

# Attach to the session (or just keep it running in the background)
# If arguments are passed, execute them, otherwise attach to the session
if [ $# -gt 0 ]; then
    exec "$@"
else
    # Keep the container running by attaching to tmux
    exec tmux attach-session -t pentest
fi
