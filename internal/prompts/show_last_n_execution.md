Capture a screenshot of the last N command executions from the Kali terminal. Useful for seeing terminal state after timeouts, during interactive sessions (msfconsole, vim, etc.), or to verify long-running command progress. Returns a PNG image of the terminal state.

## Parameters

- **n** (integer, required): Number of recent command executions to capture (default: 1)

## When to use

- After receiving a timeout to see more context
- When entering interactive modes (msfconsole, vim, etc.)
- To verify long-running command progress
- To capture TUI application states

## Example Workflows

### Monitoring a long-running scan
```
1. Run: nmap -sV -sC target.com    → May timeout
2. Run: ShowLastNExecution(n=1)    → See current scan progress
3. Wait and repeat as needed
```

### Interactive tool usage
```
1. Run: msfconsole                 → Likely times out during startup
2. Run: ShowLastNExecution(n=1)    → Verify msf> prompt appeared
3. Run: use exploit/...            → Send command
4. Run: ShowLastNExecution(n=2)    → See last 2 commands with outputs
```

## Notes

- This tool captures the visual state of the `agent` tmux window
- The screenshot is generated using the `screenshotter` tmux window
- Returns a PNG image encoded in the response
- Higher N values show more command history in a single screenshot
