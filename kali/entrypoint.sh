#!/bin/bash

# Create tmux session with agent and screenshotter windows
tmux new-session -d -s pentest -n agent
tmux new-window -t pentest -n screenshotter

# Pre-source capture script in screenshotter window
tmux send-keys -t pentest:screenshotter "source /usr/local/bin/capture_cmds.sh" Enter

# Focus agent window and attach
tmux select-window -t pentest:agent
exec tmux attach-session -t pentest
