# Tandem MCP, a Penetration Testing Engagement MCP Server

## Overview

This MCP server provides a **terminal** tool that executes commands in an isolated Kali Linux container for penetration testing engagements.

## Available Tools

### terminal

Executes penetration testing commands in a secure Kali Linux container with pre-installed security tools.

## When to Use This Server

Use the terminal tool for ALL penetration testing activities including:

### 1. Reconnaissance & Information Gathering

- **Network scanning**: Use when you need to discover hosts, open ports, or services
  - Tools: nmap, masscan, netdiscover, arp-scan
  - Example: nmap -sV -sC {target}
- **DNS enumeration**: Use when investigating domain information
  - Tools: dig, nslookup, dnsenum, fierce
- **Subdomain discovery**: Use when finding subdomains
  - Tools: sublist3r, amass, subfinder
- **Web reconnaissance**: Use when gathering web application info
  - Tools: whatweb, wafw00f, nikto

### 2. Vulnerability Scanning & Analysis

- **Automated scanning**: Use when performing comprehensive vulnerability scans
  - Tools: nmap (with NSE scripts), nikto, wpscan, nuclei
- **SSL/TLS testing**: Use when analyzing certificate and protocol security
  - Tools: sslscan, testssl.sh
- **Directory/file enumeration**: Use when discovering hidden paths
  - Tools: gobuster, dirb, dirsearch, ffuf

### 3. Web Application Testing

- **SQL injection testing**: Use when testing for database vulnerabilities
  - Tools: sqlmap
- **XSS testing**: Use when testing for cross-site scripting
  - Tools: xsser, custom curl/wget commands
- **API testing**: Use when interacting with REST/GraphQL APIs
  - Tools: curl, wget, httpie
- **Request manipulation**: Use when crafting custom HTTP requests
  - Tools: curl, wget with custom headers/methods

### 4. Password Attacks

- **Hash cracking**: Use when attempting to crack password hashes
  - Tools: john, hashcat
- **Password spraying**: Use when testing common passwords against accounts
  - Tools: hydra, medusa, ncrack
- **Wordlist generation**: Use when creating custom wordlists
  - Tools: crunch, cewl

### 5. Network Exploitation

- **Metasploit operations**: Use when launching exploits or post-exploitation
  - Tools: msfconsole, msfvenom (for payload generation)
- **Port forwarding/tunneling**: Use when pivoting through networks
  - Tools: ssh, socat, chisel
- **Packet crafting**: Use when creating custom network packets
  - Tools: hping3, scapy

### 6. Wireless Testing

- **WiFi analysis**: Use when testing wireless network security
  - Tools: aircrack-ng suite, reaver, wifite
- **Bluetooth scanning**: Use when testing Bluetooth devices
  - Tools: hcitool, bluez utilities

### 7. Post-Exploitation & Privilege Escalation

- **Linux enumeration**: Use when gathering system information
  - Tools: linpeas.sh, linux-exploit-suggester
- **File system analysis**: Use when searching for sensitive files
  - Tools: find, grep, locate
- **Process/service analysis**: Use when investigating running services
  - Tools: ps, netstat, ss, lsof

## Usage Guidelines

### Command Structure

- **command**: The primary tool or binary to execute (e.g., "nmap", "sqlmap", "curl")
- **argument**: Array of command-line arguments and flags

### Best Practices

1. **Always start with reconnaissance** before attempting exploitation
2. **Use non-intrusive scans first** (e.g., -sS instead of -sT for nmap)
3. **Verify target scope** before executing any commands
4. **Use appropriate timing** to avoid detection (e.g., nmap -T2 for stealth)
5. **Chain commands carefully**
6. **Handle sensitive data** appropriately (passwords, tokens, etc.)
7. **Use native tools for documentation** - avoid using terminal for file operations like cat, echo, grep as the agent has better native tools for those tasks

### Example Workflows

**Initial Reconnaissance:**

1. Host discovery: terminal(command: "nmap", argument: ["-sn", "192.168.1.0/24"])
2. Port scan: terminal(command: "nmap", argument: ["-p-", "-T4", "192.168.1.10"])
3. Service detection: terminal(command: "nmap", argument: ["-sV", "-sC", "-p", "80,443", "192.168.1.10"])

**Web Application Testing:**

1. Technology detection: terminal(command: "whatweb", argument: ["https://target.com"])
2. Directory enumeration: terminal(command: "gobuster", argument: ["dir", "-u", "https://target.com", "-w", "/usr/share/wordlists/dirb/common.txt"])
3. SQL injection: terminal(command: "sqlmap", argument: ["-u", "https://target.com/page?id=1", "--batch"])

**Password Attacks:**

1. Hash identification: terminal(command: "hashid", argument: ["hash_string"])
2. Crack with john: terminal(command: "john", argument: ["--wordlist=/usr/share/wordlists/rockyou.txt", "hashes.txt"])

### Important Notes

- The container has internet access for downloading resources if needed
- All standard Kali Linux tools are pre-installed and ready to use

## Security Reminders

- Follow rules of engagement strictly
