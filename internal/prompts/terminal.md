# Terminal tool description

Execute commands in a persistent Kali Linux container with Docker isolation.

**IMPORTANT BEHAVIOR:**

- Each command execution is ISOLATED (separate process, fresh shell state)
- The container FILESYSTEM is SHARED across all executions

**PERSISTS** between calls (✓):
• Files and directories created/modified
• Network connections and listening ports
• Background processes started with '&'
• Installed packages and system changes

**DOES NOT PERSIST** between calls (✗):
• Shell variables (export, set)
• Working directory (cd commands)
• Shell history
• Process environment variables

**WORKAROUNDS** for state management:
• Combine commands: use 'bash -c "cd /tmp && ls"' instead of separate cd and ls
• Save state to files: write variables/data to filesystem for later use
• Use absolute paths: specify full paths instead of relying on working directory

Example workflow:

1. Run: nmap -sV target > /tmp/scan.txt (saves to file)
2. Run: cat /tmp/scan.txt (reads persisted file)
3. Run: cd /tmp && pwd (combined command works)
4. Run: pwd (returns /root, cd didn't persist)
