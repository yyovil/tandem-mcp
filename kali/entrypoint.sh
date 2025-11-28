#!/bin/bash

tmux new-session -d -s pentest -n agent
tmux new-window -t pentest -n screenshotter
tmux send-keys -t pentest:screenshotter "source /usr/local/bin/capture_cmds.sh" Enter
tmux select-window -t pentest:agent
exec tmux attach-session -t pentest
