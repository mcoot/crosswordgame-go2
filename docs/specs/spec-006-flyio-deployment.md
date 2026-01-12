---
spec_id: "spec-006"
spec_name: "Fly.io Deployment"
status: "IN_PROGRESS"
---
# spec-006 - Fly.io Deployment

## Overview

Dockerize the crossword game application and deploy to Fly.io with GitHub Actions for automatic deployment. The deployment is optimized for minimal cost given low expected usage, targeting Fly.io's free tier.

## Relevant context

- **Application**: Go 1.25.5 web server with SSE for real-time game updates
- **Entry point**: `cmd/server/main.go`
- **Build dependency**: Templ templates must be generated before compilation
- **Static assets**: `internal/web/static/` (CSS), `data/words.txt` (dictionary ~1.9MB)
- **Port**: 8080 (hardcoded)
- **Existing CI**: `.github/workflows/ci.yml` runs tests and linting

### Cost Optimization Strategy
- Single `shared-cpu-1x` machine (256MB) - within Fly.io free tier
- Single region (Sydney) - no redundancy
- `auto_stop_machines = false` - required for SSE connections to persist
- No persistent storage (in-memory state is acceptable)

### Constraints
- SSE requires persistent connections, so machines cannot auto-stop
- Games in progress will be lost on machine restart (acceptable for this use case)

## Task implementation strategy

1. **Create Dockerfile**
   - Multi-stage build: Go builder stage → minimal Alpine runtime
   - Generate templ templates in build stage
   - Copy only required assets: binary, `data/`, `internal/web/static/`
   - Target image size: ~20-30MB

2. **Create .dockerignore**
   - Exclude `.git`, `.github`, `docs/`, `bin/`, coverage files
   - Keep `data/` directory (dictionary file required)

3. **Test Docker build locally**
   - Build image and run container
   - Verify application works at `localhost:8080`

4. **Create fly.toml configuration**
   - App name and Sydney region
   - HTTP service on port 8080 with HTTPS redirect
   - VM config: 256MB shared CPU
   - Disable auto-stop (SSE requirement)

5. **Create GitHub Actions deploy workflow**
   - Trigger on push to master and manual dispatch
   - Run CI checks (format, lint, tests) as prerequisite
   - Deploy to Fly.io only after CI passes
   - Use `FLY_API_TOKEN` secret for authentication

6. **Initial Fly.io setup (manual)**
   - Install flyctl CLI
   - Create Fly.io account and app
   - Generate deploy token
   - Add `FLY_API_TOKEN` secret to GitHub repository

7. **Verify deployment**
   - Push to master, confirm CI → Deploy pipeline
   - Test live application functionality
   - Check Fly.io dashboard for resource usage

## Files to create

| File | Purpose |
|------|---------|
| `Dockerfile` | Multi-stage build for minimal production image |
| `.dockerignore` | Exclude unnecessary files from Docker context |
| `fly.toml` | Fly.io application configuration |
| `.github/workflows/deploy.yml` | CI-gated auto-deployment workflow |

## Status details

Status: IN_PROGRESS

**Completed tasks:**
- [x] Task 1: Create Dockerfile (multi-stage build, ~32MB image)
- [x] Task 2: Create .dockerignore
- [x] Task 3: Test Docker build locally (verified working)
- [x] Task 4: Create fly.toml configuration
- [x] Task 5: Create GitHub Actions deploy workflow
- [x] Added Taskfile tasks: `docker:build`, `docker:run`, `docker:run:detached`, `docker:stop`

**Remaining manual steps:**
- [ ] Task 6: Install flyctl CLI (`brew install flyctl`)
- [ ] Task 6: Login to Fly.io (`flyctl auth login`)
- [ ] Task 6: Create Fly.io app (`flyctl launch --no-deploy` or `flyctl apps create crosswordgame-go`)
- [ ] Task 6: Generate deploy token (`flyctl tokens create deploy`)
- [ ] Task 6: Add `FLY_API_TOKEN` secret to GitHub repository
- [ ] Task 7: Deploy and verify
