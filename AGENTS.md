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

- **Store layer**: `pkg/store/` defines interfaces (`AccountRepo`, `CharacterRepo`, `WorldRepo`). Implementations: `internal/store/mongostore` (MongoDB), `internal/store/mysqldb` (stub).
- **Auth server**: `pkg/summit/auth/` — SRP6 auth, realm list, gRPC management API.
- **World server**: `pkg/summit/world/` — game session handling, packet handlers, object manager.
- **WoW protocol**: `pkg/wow/` — packet types, opcodes (generated), SRP6 crypto, player/object models.
- **Proto**: `proto/auth/v1/` → generated to `pkg/pb/proto/`.
- **Config**: `summit.yaml` (runtime config).
- **Web Client**: `client/` — browser-based WoW 3.3.5a client using React, Three.js, Zustand, and WebSockets (see [client/AGENTS.md](file:///Users/paalgyula/Workspace/wow/summit/client/AGENTS.md)).

## Linting

golangci-lint. Config `.golangci.yaml`: skips `*_string.go` files and tests. Presets enabled: bugs, comment, complexity, error, format, import, metalinter, module, performance, sql, style, test, unused. Disabled: varnamelen, asasalint, depguard, containedctx, gomnd, godox.

## Testing

- Tests live alongside source (`*_test.go`).
- No test infrastructure prerequisites found — tests are unit-level.
- Run a single test: `go test -v ./pkg/summit/auth/... -run TestLoginChallenge`

## Reference Implementations — ALWAYS CHECK FIRST

**When debugging, implementing, or fixing ANY feature, you MUST first consult the reference implementations:**

| Reference | Path | Purpose |
|-----------|------|---------|
| **TrinityCore (TC)** | `/Users/paalgyula/Workspace/wow/trinitycore` | Primary reference — most complete WotLK implementation |
| **AzerothCore (AC)** | `/Users/paalgyula/Workspace/wow/azerothcore-wotlk` | Secondary reference — TC fork with same architecture |

### Workflow for ANY feature:

1. **Search TC/AC first** — find the exact packet handlers, opcodes, structures, and flow
2. **Document the packet structure** — field names, types, order, opcodes (hex)
3. **Document the handler logic** — pre-conditions, post-conditions, what gets broadcast
4. **Document SMSG_UPDATE_OBJECT fields** — which fields change, when, and why
5. **Implement in Summit** — match the packet structures EXACTLY (3.3.5a protocol)
6. **Verify** — all types, opcodes, and field order must match 3.3.5a exactly

### Key reference files to check:

| Area | TC Path | AC Path |
|------|---------|---------|
| Packet handlers | `src/server/game/Handlers/` | `src/server/game/Handlers/` |
| Player death/resurrect | `src/server/game/Entities/Player/Player.cpp` | `src/server/game/Entities/Player/Player.cpp` |
| Unit death | `src/server/game/Entities/Unit/Unit.cpp` | `src/server/game/Entities/Unit/Unit.cpp` |
| Spirit healer | `src/server/game/Handlers/NPCHandler.cpp` | `src/server/game/Handlers/NPCHandler.cpp` |
| Misc handlers (death) | `src/server/game/Handlers/MiscHandler.cpp` | `src/server/game/Handlers/MiscHandler.cpp` |
| Spells (resurrect) | `src/server/game/Spells/Spell.cpp`, `SpellEffects.cpp` | `src/server/game/Spells/Spell.cpp`, `SpellEffects.cpp` |
| Opcodes | `src/server/game/Server/Protocol/Opcodes.h` | `src/server/game/Server/Protocol/Opcodes.h` |
| Packet structures | `src/server/game/Server/Packets/MiscPackets.h` | `src/server/game/Server/Packets/MiscPackets.h` |

### When to check:

- **EVERY new feature implementation** — find TC/AC equivalent first
- **EVERY bug fix** — understand what TC/AC does differently
- **EVERY packet handler** — verify opcode, structure, and flow match
- **EVERY update object** — verify field indices and types match 3.3.5a

### Existing reference docs:

| Document | Path | Purpose |
|----------|------|---------|
| Death/Resurrect System | `docs/death-resurrect-system.md` | Complete death flow, packets, spirit healer, corpse, durability |

## Gotchas

- `go.mod` declares `go 1.25.0` but CI uses `1.21.0` — version mismatch is intentional per codebase state.
- `docs/version.go` defines build info vars (`Version`, `Branch`, `Gitsha`, etc.) injected via `-ldflags` in Makefile.
- Store init in `cmd/summit` uses `mongostore.Connect()` with MongoDB URI from config.
- `//nolint:all` appears on entry point files — don't add more of these unless intentional.
- `cmd/worldbaby/` and `cmd/summitbot/` are dev/test tools, not production binaries.
