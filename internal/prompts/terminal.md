# Terminal tool description

Execute commands in a persistent Kali Linux container with Docker isolation and tmux session management.

## Execution Behavior

Commands are executed in the `agent` window of a tmux session named `pentest`. The tool waits up to **2 minutes** for command completion.

### Completion Detection
- The tool monitors for the `KALI[path]# ` prompt to detect when a command has finished
- If the prompt reappears within 2 minutes: returns **text output**
- If timeout occurs: returns a **screenshot image** of the current terminal state

### Timeout Handling
When a command times out (takes longer than 2 minutes), you will receive:
1. A message indicating the timeout
2. A PNG screenshot of the terminal's current visual state

This allows you to see what's happening and decide how to proceed.

## Visual State Capture

Use `showLastNExecution N` to capture screenshots of the last N command executions:

```bash
showLastNExecution 1   # Capture last command and its output
showLastNExecution 3   # Capture last 3 commands and their outputs
showLastNExecution 5   # Capture last 5 commands
```

**When to use:**
- After receiving a timeout to see more context
- When entering interactive modes (msfconsole, vim, etc.)
- To verify long-running command progress
- To capture TUI application states

## Session Architecture

```
Container (kali:headless)
└── tmux session: pentest
    ├── Window: agent        ← Commands executed here
    └── Window: screenshotter ← Visual capture runs here
```

**PERSISTS** between calls (✓):
• Files and directories created/modified
• Working directory (cd commands persist!)
• Environment variables (export persists!)
• Network connections and listening ports
• Background processes started with '&'
• Installed packages and system changes
• Shell history within the session

**Session Isolation:**
• Single persistent bash shell in the agent window
• State carries across all command calls
• Container filesystem shared

## Interactive Commands

For interactive tools (msfconsole, vim, nmap with prompts, etc.):

1. Start the tool - it may timeout if waiting for input
2. Use `showLastNExecution 1` to see the current state
3. Send appropriate input/commands based on what you see
4. Repeat as needed for multi-step interactions

Example workflow for msfconsole:
```
1. Run: msfconsole           → May timeout (startup takes time)
2. Run: showLastNExecution 1 → See if msf prompt appeared
3. Run: use exploit/...      → Send exploit command
4. Run: showLastNExecution 1 → Verify state
```

## Best Practices

1. **Long-running scans**: Expect timeout, use visual capture to monitor
2. **Interactive tools**: Send commands incrementally, verify with screenshots
3. **Background processes**: Use `&` and `disown` for truly detached processes
4. **State verification**: Use `showLastNExecution` liberally to understand current state
5. **Absolute paths**: Still recommended for reliability
