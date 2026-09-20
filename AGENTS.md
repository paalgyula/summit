# Summit - Agent Instructions

## Project Overview

Go-based WoW 3.3.5a (WotLK) server emulator. Three main binaries + supporting tools.

## Binaries (`cmd/`)

| Binary | Purpose | Default Listen |
|--------|---------|----------------|
| `summit` | Auth server + world server combined | auth: `:5000`, world: `:5002` |
| `serworm` | Proxy worm for packet capture | `:5000`, proxies to real auth |
| `datagen` | DBC converter + opcode/header codegen | N/A (CLI tool) |
| `summitbot` | Test client for auth+world login flow | N/A (CLI tool) |
| `summitctl` | CLI management utility | N/A |
| `worldbaby` | Dev test harness for world packets | N/A |

## Build & Verify Commands

```bash
make              # build all three main binaries to bin/
make build-dist   # cross-compiled production build (CGO_ENABLED=0)
make lint         # golangci-lint run ./...
go test -v ./...  # run all tests
go build -o bin/summit cmd/summit/summit.go   # single binary
```

**CI order** (`.github/workflows/go.yml`): lint → build → test

## Code Generation

```bash
make deps         # install protoc-gen-go, stringer, impl
make gen          # generate store interface stubs + go generate
go generate ./... # runs all go:generate directives
```

**Protobuf** (`buf.yaml`): schemas in `proto/auth/v1/`, generated Go → `pkg/pb/proto/`. Use `buf generate` (not raw protoc).

**Opcode generation**: `datagen opcodes` fetches and parses WoW opcode headers.

**Interface stubs**: `make gen` uses `impl` to scaffold store and service interfaces.

## Configuration

Viper-based. Config file: `server.yaml` (or `summit.yaml` in repo root). Env prefix: `SUMMIT_`. Defaults in `cmd/summit/summit.go:20-44`.

## Architecture

- **Store layer**: `pkg/store/` defines interfaces (`AccountRepo`, `CharacterRepo`, `WorldRepo`). Implementations: `internal/store/localdb` (YAML-based, uses `summit-store.yaml`), `internal/store/mysqldb`.
- **Auth server**: `pkg/summit/auth/` — SRP6 auth, realm list, gRPC management API.
- **World server**: `pkg/summit/world/` — game session handling, packet handlers, object manager.
- **WoW protocol**: `pkg/wow/` — packet types, opcodes (generated), SRP6 crypto, player/object models.
- **Proto**: `proto/auth/v1/` → generated to `pkg/pb/proto/`.
- **Config**: `summit.yaml` (runtime config), `summit-store.yaml` (account/character data).

## Linting

golangci-lint. Config `.golangci.yaml`: skips `*_string.go` files and tests. Presets enabled: bugs, comment, complexity, error, format, import, metalinter, module, performance, sql, style, test, unused. Disabled: varnamelen, asasalint, depguard, containedctx, gomnd, godox.

## Testing

- Tests live alongside source (`*_test.go`).
- No test infrastructure prerequisites found — tests are unit-level.
- Run a single test: `go test -v ./pkg/summit/auth/... -run TestLoginChallenge`

## Gotchas

- `go.mod` declares `go 1.25.0` but CI uses `1.21.0` — version mismatch is intentional per codebase state.
- `docs/version.go` defines build info vars (`Version`, `Branch`, `Gitsha`, etc.) injected via `-ldflags` in Makefile.
- Store init in `cmd/summit` uses `localdb.InitYamlDatabase("summit-store.yaml")` — YAML file must exist.
- `//nolint:all` appears on entry point files — don't add more of these unless intentional.
- `cmd/worldbaby/` and `cmd/summitbot/` are dev/test tools, not production binaries.
