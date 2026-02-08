---
spec_id: "spec-008"
spec_name: "LLM Bot Strategy"
status: "ACTIVE"
---
# spec-008 - LLM Bot Strategy

## Overview

Adds an LLM-driven bot strategy using Gemini 2.5 Flash-Lite that plays meaningfully by picking useful letters and placing them to form words. The strategy falls back to random on any LLM failure, rate limit, or invalid response, ensuring robustness.

Key design goals:
- Cost control: daily call limit (default 5000), 3-second timeouts, compact prompts (~100 tokens input)
- Reliability: every failure path falls back to `RandomStrategy`
- Testability: LLM client behind a dependency interface with mock for unit tests

## Relevant context

- Builds on spec-007 (Bot Players) which established the `Strategy` interface and `RandomStrategy`
- Targets Fly.io free tier (256MB RAM) and Gemini free tier (1,000 RPD for paid, higher for free)
- Uses `google.golang.org/genai` SDK (already a transitive dependency)
- The `context.Context` parameter was added to the `Strategy` interface to support LLM call timeouts

### Architecture decisions

- **Interface change**: Added `context.Context` to `Strategy.ChooseLetter` and `Strategy.ChoosePosition` since Go idiom requires context for cancellable/timed operations
- **Daily limit counter**: In-memory with mutex, resets on UTC date change via `clock.Clock` (testable). Acceptable since the counter resets on process restart anyway
- **No board state in letter prompt**: During letter selection, the bot doesn't have access to its board in a way that's useful (the letter hasn't been announced yet), so the prompt uses game context only
- **Code fence stripping**: LLMs commonly wrap JSON in markdown code fences even when told not to; the parser handles this
- **Graceful degradation**: LLM strategy shows in UI dropdown even without API key configured; attempting to add such a bot fails with "unknown bot strategy" at the strategy map lookup

## Task implementation strategy

1. Add `context.Context` to `Strategy` interface and update all implementations/call sites
2. Create `llm.Client` dependency interface, `GeminiClient` implementation, and `MockLLMClient`
3. Create `LLMStrategy` with prompts, JSON parsing, daily limit counter, and fallback logic
4. Add `BotStrategyLLM` model constant and update `ValidBotStrategies()`
5. Wire into factory: `Config` fields, conditional `GeminiClient` creation, `newWithDependencies` params
6. Update `main.go` to read `GEMINI_API_KEY` and `GEMINI_MODEL` env vars
7. Write comprehensive unit tests for all success and fallback paths
8. Run `task generate`, `task test`, `task lint` to verify

## Status details

All tasks complete. Implementation verified with:
- 18 new LLM strategy unit tests (all passing)
- All existing tests passing
- Lint clean (0 issues)

### Files added
- `internal/dependencies/llm/client.go` - Client interface and Response type
- `internal/dependencies/llm/gemini.go` - Gemini SDK wrapper
- `internal/dependencies/mocks/llm.go` - Mock for testing
- `internal/services/bot/llm_strategy.go` - LLM strategy implementation
- `internal/services/bot/llm_strategy_test.go` - Tests

### Files modified
- `internal/services/bot/strategy.go` - Added context.Context params
- `internal/services/bot/random_strategy.go` - Added context.Context params
- `internal/services/bot/service.go` - Pass ctx at call sites
- `internal/services/bot/strategy_test.go` - Added context.Background() to calls
- `internal/model/bot_strategy.go` - Added LLM constant
- `internal/factory/factory.go` - Config fields, LLM wiring
- `internal/factory/test_factory.go` - Updated call signature, added NewTestAppWithLLM
- `cmd/server/main.go` - Env var reading
- `go.mod` / `go.sum` - genai dependency upgrade

### Deployment
Set `GEMINI_API_KEY` as a Fly.io secret (`fly secrets set GEMINI_API_KEY=<key>`). Optionally set `GEMINI_MODEL` env var to override the default `gemini-2.5-flash-lite`.
