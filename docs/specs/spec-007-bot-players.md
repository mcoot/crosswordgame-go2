---
spec_id: "spec-007"
spec_name: "Bot Players"
status: "ACTIVE"
---
# spec-007 - Bot Players

## Overview

Add bot players that the game host can add to a lobby. Bots participate fully in the game: they announce random letters (A-Z) when it's their turn and place on random empty cells. A `Strategy` interface allows future smarter bot implementations.

Scope: backend core logic + JSON API + web handlers + tests. Web UI (buttons/forms) deferred to follow-up.

## Relevant context

- `Player.IsBot` field mirrors the existing `IsGuest` pattern
- Bot service wraps lobby controller for add/remove (validates host, creates bot player, delegates to JoinLobby/LeaveLobby)
- `ProcessBotActions` implements a cascading loop: if all remaining actions in the current state are bot actions, they chain automatically
- Bots participate in announcer rotation like normal players
- `ProcessBotActions` returns `[]BotAction` so handlers can emit SSE broadcasts

### Key files

- `internal/services/bot/` - Strategy interface, RandomStrategy, Service
- `internal/model/player.go` - `IsBot` field added to Player
- `internal/model/errors.go` - `ErrNotBot` error added
- `internal/api/handler/lobby.go` - AddBot/RemoveBot endpoints
- `internal/api/handler/game.go` - ProcessBotActions after Start/Announce/Place
- `internal/web/handler/game.go` - ProcessBotActions after Start/Announce/Place
- `internal/web/handler/lobby.go` - AddBot/RemoveBot web handlers
- `internal/factory/factory.go` - BotService wired into App

### API endpoints

- `POST /api/v1/lobbies/{code}/bots` - Add bot to lobby (host only)
- `DELETE /api/v1/lobbies/{code}/bots/{player_id}` - Remove bot from lobby (host only)

### Web endpoints

- `POST /lobby/{code}/bots/add` - Add bot (form)
- `POST /lobby/{code}/bots/remove` - Remove bot (form with bot_player_id)

## Task implementation strategy

1. Model changes: Add `IsBot` to Player, `ErrNotBot` to errors
2. Bot strategy interface + RandomStrategy implementation with unit tests
3. Bot service: CreateBotPlayer, AddBotToLobby, RemoveBotFromLobby with tests
4. Bot service: ProcessBotActions cascading loop with tests
5. Response/request type updates (IsBot in Player/LobbyMember responses)
6. API handler + router changes (AddBot/RemoveBot endpoints, ProcessBotActions after game actions)
7. Web handler integration (ProcessBotActions, AddBot/RemoveBot web handlers)
8. Factory wiring (BotService in App, TestApp)
9. Integration tests (bot game flows)
10. API E2E tests (bot endpoints and game-with-bot flows)

## Status details

All tasks complete. Implementation includes:
- 18 bot service unit tests (all passing)
- 4 integration tests (bot game flow, all bots game, add/remove bot, bot in lobby starts game)
- 3 API tests (add bot, remove bot, game with bot)
- All existing tests continue to pass
- Zero lint issues
