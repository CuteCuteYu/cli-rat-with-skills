# Command Distribution System

A client-server architecture command distribution system developed in Go. The server can manage multiple clients, send system commands to specified clients, and retrieve execution results.

## Features

- **Multi-Client Management**: Support for managing multiple client connections simultaneously
- **Command Distribution**: Send commands to specified clients and execute remotely
- **Result Return**: Clients return command execution results to the server
- **Command History**: Save command execution history with ability to query command details
- **Background Execution**: Server supports running as a daemon process
- **Forced Disconnection**: Server can actively disconnect specified client connections

## System Architecture

```
┌─────────────┐                          ┌─────────────┐
│   Server    │                          │   Client    │
│  :8080      │                          │             │
├─────────────┤                          ├─────────────┤
│             │  HTTP JSON               │             │
│  HTTP Server│◄─────────────────────────┤  Poll Loop  │
│             │       1 sec/poll         │             │
│             │                          ├─────────────┤
│ Command Q   │                          │   cmd.exe   │
│ Session Mgr │─── send command ────────►│   /c        │
│             │                          │             │
└─────────────┘◄── return result ───────└─────────────┘
```

**Workflow:**
1. Client registers with the server upon startup and receives a unique ClientID
2. Client polls the server every second to retrieve pending commands
3. Server enqueues commands via the `send` command
4. Client executes commands via `cmd.exe /c`
5. Execution results are returned to the server via the `/result` endpoint
6. Server saves command history and execution results

## Requirements

- Go 1.26+
- Windows operating system
- Local area network or local execution environment

## Installation and Compilation

### Prerequisites

1. **Install Go Environment**

   Download and install Go 1.26 or higher:
   - Visit [https://golang.org/dl/](https://golang.org/dl/)
   - Download the Windows installer and install
   - Verify installation:
     ```powershell
     go version
     ```

2. **Configure Environment Variables (Optional)**

   If you need to customize GOPATH:
   ```powershell
   # Set GOPATH
   $env:GOPATH = "C:\Go\Projects"

   # Add to PATH
   $env:PATH += ";$env:GOPATH\bin"
   ```

### Compilation Steps

#### Method 1: Using Build Script (Recommended)

```powershell
# Navigate to project directory
cd C:\Users\j4543\Desktop\test\NEWTEST

# Execute build script
powershell -File build.ps1
```

The build script will automatically compile both server and client, and display build results:
```
[OK] Server built successfully
[OK] Client built successfully

Build complete!
```

#### Method 2: Manual Compilation

```powershell
# Navigate to project directory
cd C:\Users\j4543\Desktop\test\NEWTEST

# Compile server (includes main.go and command.go)
go build -o server.exe server/main.go server/command.go

# Compile client
go build -o client.exe client/main.go
```

#### Method 3: Cross-Compilation

For compiling to other platforms:

```powershell
# Linux 64-bit
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o server-linux server/main.go server/command.go
go build -o client-linux client/main.go

# macOS 64-bit
$env:GOOS = "darwin"
$env:GOARCH = "amd64"
go build -o server-mac server/main.go server/command.go
go build -o client-mac client/main.go

# Restore Windows compilation
$env:GOOS = "windows"
$env:GOARCH = "amd64"
```

#### Compilation Optimization Options

```powershell
# Reduce executable file size
go build -ldflags "-s -w" -o server.exe server/main.go server/command.go
go build -ldflags "-s -w" -o client.exe client/main.go

# Full optimization (recommended for release)
go build -ldflags "-s -w" -trimpath -o server.exe server/main.go server/command.go
go build -ldflags "-s -w" -trimpath -o client.exe client/main.go
```

### Build Artifacts

After compilation, the following files are generated:

| File | Size (approx) | Description |
|------|---------------|-------------|
| `server.exe` | ~2 MB | Server program |
| `client.exe` | ~2 MB | Client program |

### Verify Compilation Results

```powershell
# View server help
server help

# View client version (displayed on startup)
.\client.exe
```

## Usage Instructions

### Environment Configuration (Recommended)

Add `server.exe` to the system PATH environment variable to use the `server` command from any directory.

#### Adding to PATH on Windows

**Method 1: Temporary Addition (Current Session)**

```powershell
# Add to current session PATH
$env:PATH += ";C:\Users\j4543\Desktop\test\NEWTEST"

# Verify
server help
```

**Method 2: Permanent Addition (PowerShell)**

```powershell
# Add to user PATH (permanent)
[Environment]::SetEnvironmentVariable("Path", $env:PATH + ";C:\Users\j4543\Desktop\test\NEWTEST", "User")

# Refresh environment variables
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","User")
```

**Method 3: Graphical Interface Addition**

1. Right-click "This PC" → "Properties"
2. "Advanced system settings" → "Environment Variables"
3. Find `Path` under "User variables", click "Edit"
4. Click "New", add `C:\Users\j4543\Desktop\test\NEWTEST`
5. Confirm and save, reopen terminal

#### Claude Code Skill Usage

This project includes the `c2-server` skill, allowing you to operate the server using natural language in Claude Code.

##### Prerequisites

⚠️ **Before using this skill, you must complete the following steps:**

1. **Compile the project** - Generate the `server.exe` binary file
2. **Add to PATH** - Add the directory containing `server.exe` to your system environment variables

**Detailed steps:**

```powershell
# Step 1: Compile the project (replace with your actual project path)
cd <项目路径>
powershell -File build.ps1

# Step 2: Add to environment variables (permanent, replace with your actual path)
[Environment]::SetEnvironmentVariable("Path", $env:PATH + ";<项目路径>", "User")
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","User")

# Step 3: Verify installation
server help
```

**Or use current directory (simplest):**

```powershell
# Execute in project directory, automatically uses current path
[Environment]::SetEnvironmentVariable("Path", $env:PATH + ";$PWD", "User")
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","User")
server help
```

If the `server help` command executes successfully, the configuration is correct and you can proceed with skill installation.

##### Skill File Location

The skill file is located in the project root directory:
```
.claude/skills/c2-server/SKILL.md
```

##### Installing the Skill

**Method 1: Copy to Claude Code skills directory (Recommended)**

```powershell
# Create Claude Code skills directory (if it doesn't exist)
mkdir -p ~/.claude/skills/c2-server

# Copy skill file
copy .claude\skills\c2-server\SKILL.md ~/.claude/skills/c2-server/SKILL.md
```

**Method 2: Symbolic Link (Easier for development updates)**

```powershell
# On Windows using mklink (replace with your actual path)
mklink /D "%USERPROFILE%\.claude\skills\c2-server" "<项目路径>\.claude\skills\c2-server"
```

##### Using the Skill

After installation, you can operate the server using natural language in Claude Code:

```
You: Start the server
Claude: server -init

You: View online clients
Claude: server list

You: Send ipconfig command to client-xxx
Claude: server send client-xxx "ipconfig"

You: View command history
Claude: server history

You: Disconnect client client-xxx
Claude: server kill client-xxx
```

##### Supported Operations

- Start/Stop server
- View online client list
- Send commands to specified clients
- View command execution history
- View command details and results
- Disconnect client connections

##### Skill File Content

The complete skill file is located at `.claude/skills/c2-server/SKILL.md`, containing detailed instructions and examples for all commands.

### Server Commands

#### Start/Stop Server

```powershell
# After adding to PATH (recommended)
server -init

# Or use full path (when not added to PATH)
.\server.exe -init

# Custom listen address and port
server -init -ip 0.0.0.0 -port 9090

# Stop server
server stop

# Check server status
server status
```

#### Client Management

```powershell
# View online client list
server list

# Disconnect specified client connection
server kill <client_id>
```

#### Command Operations

```powershell
# Send command to client
server send <client_id> <command>

# Example: View client directory
server send client-20260403013011 "dir"

# Example: Execute PowerShell command
server send client-20260403013011 "powershell Get-Process"

# Example: View system information
server send client-20260403013011 "ipconfig /all"
```

#### Command History

```powershell
# View command history (recent 10 entries)
server history

# View specified command details
server show <command_id>

# Example
server show cmd-20260403013032
```

#### Help Information

```powershell
server help
```

### Client Usage

#### Modify Server Address

Edit the `serverAddr` variable in `client/main.go`:

```go
var (
    serverAddr = "http://192.168.1.100:8080"  // Change to server address
    // ...
)
```

#### Start Client

```powershell
.\client.exe
```

After startup, the client will:
1. Register with the server and receive a ClientID
2. Save ClientID to `client.id` file, automatically restored on restart
3. Poll the server every second for commands
4. Execute received commands and return results
5. Press Enter to exit

## Configuration Files

The server generates the following files in the current directory during runtime:

| File | Description |
|------|-------------|
| `session.json` | Client session information |
| `commands.json` | Command execution history |
| `server.pid` | Server process ID |
| `server.lock` | Server lock file |
| `server.log` | Server runtime log |

Files generated by the client during runtime:

| File | Description |
|------|-------------|
| `client.id` | Client registration ID |

## API Endpoints

### POST /register

Client registration

**Request:**
```json
{
  "name": "client"
}
```

**Response:**
```json
{
  "client_id": "client-20260403013011",
  "status": "registered"
}
```

### GET /poll?client_id=xxx

Client command polling

**Response (no command):**
```json
{
  "status": "no_command"
}
```

**Response (command available):**
```json
{
  "id": "cmd-20260403013032",
  "content": "dir",
  "client_id": "client-20260403013011",
  "status": "pending"
}
```

### POST /result

Submit command execution result

**Request:**
```json
{
  "command_id": "cmd-20260403013032",
  "status": "completed",
  "result": "Execution result..."
}
```

**Response:**
```json
{
  "status": "success"
}
```

### POST /unregister

Client unregistration

**Request:**
```json
{
  "client_id": "client-20260403013011"
}
```

## Important Notes

1. **Command Execution**: Client uses `cmd.exe /c` to execute commands, supporting Windows built-in commands (such as `dir`, `cd`) and executable files
2. **Forced Exit**: Using the `kill` command sends a `__EXIT__` signal to the client, which will actively exit upon receiving it
3. **Command History**: Maximum of 10 command records are saved
4. **Network Requirements**: Client must be able to access the server's HTTP port
5. **Client Management**: When client exits, it should actively call the unregister endpoint; otherwise, use the `kill` command to manually disconnect

## Security Notice

This system is designed for use in controlled environments within a local area network. When used in production environments, consider:
- Adding authentication mechanisms
- Using HTTPS for encrypted communication
- Restricting command execution permissions
- Adding command whitelist/blacklist

## Project Structure

```
NEWTEST/
├── server/
│   ├── main.go      # Server main program
│   └── command.go   # Command management module
├── client/
│   └── main.go      # Client program
├── build.ps1        # Build script
├── go.mod           # Go module configuration
└── README.md        # This file
```

## License

This project is for learning and research purposes only.
