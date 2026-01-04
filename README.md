# crosswordgame-go2

A multiplayer word game where players take turns announcing letters and placing them on their grids to form words.

## Development

This project uses [Task](https://taskfile.dev) for build automation. Install it locally or use `go run`:

```bash
# With task installed
task check

# Without task installed
go run github.com/go-task/task/v3/cmd/task check
```

### Available Tasks

| Task | Description |
|------|-------------|
| `task` | Run all checks (test + lint) |
| `task test` | Run tests |
| `task lint` | Run linter |
| `task lint:fix` | Run linter with auto-fix |
| `task fmt` | Format code |
| `task check` | Full CI checks (fmt, lint, test) |

### Requirements

- Go 1.25+
