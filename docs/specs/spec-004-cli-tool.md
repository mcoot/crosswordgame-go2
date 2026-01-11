---
spec_id: "spec-004"
spec_name: "CLI Tool for API Interaction"
status: "PROPOSED"
---
# spec-004 - CLI Tool for API Interaction

## Overview

This specification defines a CLI tool (`cwgame`) for interacting with the crossword game JSON API and SSE events. The tool enables:

- Manual testing and debugging without writing ad-hoc curl scripts
- Automated testing scenarios via shell scripts
- Monitoring SSE events in real-time for debugging
- Quick API exploration during development

The CLI uses Cobra for command structure and follows the existing codebase conventions.

## Relevant context

- Depends on: spec-002 (JSON API endpoints and authentication)
- The JSON API base URL is `/api/v1/`
- SSE events are available at `/lobby/{code}/events` (web router)
- Authentication uses `Authorization: Bearer <token>` header
- Cobra is already in go.mod (transitive dependency via spf13/cobra)

### API Endpoints (from spec-002)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/players/guest` | POST | Create guest player |
| `/api/v1/players/register` | POST | Register player |
| `/api/v1/players/login` | POST | Login |
| `/api/v1/players/me` | GET | Get current player |
| `/api/v1/lobbies` | POST | Create lobby |
| `/api/v1/lobbies/{code}` | GET | Get lobby |
| `/api/v1/lobbies/{code}/join` | POST | Join lobby |
| `/api/v1/lobbies/{code}/leave` | POST | Leave lobby |
| `/api/v1/lobbies/{code}/config` | PATCH | Update config |
| `/api/v1/lobbies/{code}/game` | POST | Start game |
| `/api/v1/lobbies/{code}/game` | GET | Get game state |
| `/api/v1/lobbies/{code}/game` | DELETE | Abandon game |
| `/api/v1/lobbies/{code}/game/announce` | POST | Announce letter |
| `/api/v1/lobbies/{code}/game/place` | POST | Place letter |
| `/api/v1/health` | GET | Health check |

### SSE Events (from internal/web/sse/broadcaster.go)

| Event | Description |
|-------|-------------|
| `member-update` | Lobby member list changed |
| `controls-update` | Lobby controls changed |
| `game-started` | Game has started |
| `game-update` | Game status changed |
| `letter-announced` | New letter announced |
| `placement-update` | Player placed letter |
| `turn-complete` | All players placed, new turn |
| `game-complete` | Game finished |
| `game-abandoned` | Game was abandoned |
| `refresh` | Generic refresh signal |

## Design

### Command Structure

```
cwgame
├── player
│   ├── guest --name <name>           # Create guest, save token
│   ├── register --name --user --pass # Register, save token
│   ├── login --user --pass           # Login, save token
│   └── me                            # Show current player
├── lobby
│   ├── create [--grid-size <n>]      # Create lobby
│   ├── get <code>                    # Get lobby details
│   ├── join <code>                   # Join lobby
│   ├── leave <code>                  # Leave lobby
│   └── config <code> --grid-size <n> # Update config
├── game
│   ├── start <code>                  # Start game
│   ├── get <code>                    # Get game state
│   ├── announce <code> <letter>      # Announce letter
│   ├── place <code> <row> <col>      # Place letter
│   └── abandon <code>                # Abandon game
├── events <code>                     # Stream SSE events
└── health                            # Health check
```

### Global Flags

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--server` | `CWGAME_SERVER` | `http://localhost:8080` | Server URL |
| `--token` | `CWGAME_TOKEN` | (from file) | Session token |
| `--token-file` | `CWGAME_TOKEN_FILE` | `~/.cwgame/token` | Token file path |
| `--output` | - | `text` | Output format: `text`, `json` |
| `--verbose` | - | false | Verbose output |

### Token Management

Session tokens are automatically saved after `player guest`, `player register`, or `player login`:
- Default location: `~/.cwgame/token`
- Override with `--token-file` or `CWGAME_TOKEN_FILE`
- Direct token via `--token` or `CWGAME_TOKEN` takes precedence

### Output Formats

**Text mode** (default): Human-readable output
```
$ cwgame player me
Player: Alice (p_abc123)
Guest: true
```

**JSON mode**: Machine-readable output for scripting
```
$ cwgame --output json player me
{"id":"p_abc123","display_name":"Alice","is_guest":true}
```

### SSE Event Streaming

The `events` command connects to the SSE endpoint and streams events:

```
$ cwgame events ABC123
Connected to lobby ABC123
[2024-01-15 10:30:45] member-update: <html data>
[2024-01-15 10:30:52] game-started: started
[2024-01-15 10:31:01] letter-announced: T
^C
Disconnected
```

With `--json` flag, events are output as JSON lines:
```
$ cwgame events ABC123 --json
{"time":"2024-01-15T10:30:45Z","event":"member-update","data":"<html>..."}
{"time":"2024-01-15T10:30:52Z","event":"game-started","data":"started"}
```

## Package Structure

```
cmd/
├── server/
│   └── main.go              # Existing server entry point
└── cwgame/
    └── main.go              # CLI entry point

internal/
└── cli/
    ├── root.go              # Root command, global flags
    ├── client.go            # HTTP client wrapper
    ├── config.go            # Config/token management
    ├── output.go            # Output formatting (text/json)
    ├── player.go            # Player commands
    ├── lobby.go             # Lobby commands
    ├── game.go              # Game commands
    ├── events.go            # SSE event streaming
    └── health.go            # Health command
```

## Implementation Details

### HTTP Client

The client wrapper handles:
- Base URL configuration
- Token injection via `Authorization: Bearer` header
- JSON request/response marshaling
- Error response parsing

```go
type Client struct {
    baseURL    string
    httpClient *http.Client
    token      string
}

func (c *Client) Do(method, path string, body, result any) error
func (c *Client) Get(path string, result any) error
func (c *Client) Post(path string, body, result any) error
func (c *Client) Patch(path string, body, result any) error
func (c *Client) Delete(path string) error
```

### SSE Client

For event streaming, use a simple SSE parser:

```go
func (c *Client) StreamEvents(ctx context.Context, lobbyCode string, handler func(event, data string)) error
```

The SSE format is:
```
event: <event-name>
data: <line1>
data: <line2>

```

### Error Handling

API errors are displayed in a consistent format:
```
$ cwgame lobby get INVALID
Error: Lobby with code 'INVALID' not found (LOBBY_NOT_FOUND)
```

With `--json`:
```json
{"error":{"code":"LOBBY_NOT_FOUND","message":"Lobby with code 'INVALID' not found"}}
```

## Task Implementation Strategy

1. **Set up CLI skeleton** (`cmd/cwgame/main.go`, `internal/cli/root.go`)
   - Root command with global flags
   - Config and token file management
   - Cobra command tree structure

2. **Implement HTTP client** (`internal/cli/client.go`)
   - Request/response handling
   - Token injection
   - Error parsing

3. **Implement output formatting** (`internal/cli/output.go`)
   - Text and JSON formatters
   - Consistent error display

4. **Implement player commands** (`internal/cli/player.go`)
   - `player guest`, `player register`, `player login`, `player me`
   - Automatic token saving

5. **Implement lobby commands** (`internal/cli/lobby.go`)
   - `lobby create`, `lobby get`, `lobby join`, `lobby leave`, `lobby config`

6. **Implement game commands** (`internal/cli/game.go`)
   - `game start`, `game get`, `game announce`, `game place`, `game abandon`

7. **Implement SSE streaming** (`internal/cli/events.go`)
   - `events <code>` command
   - Event parsing and display

8. **Implement health command** (`internal/cli/health.go`)
   - Simple health check

9. **Add Taskfile entry**
   - `task cli:build` to build the CLI
   - Consider `task cli:run -- <args>` for quick testing

## Status Details

Status: PROPOSED - Awaiting approval before implementation.
