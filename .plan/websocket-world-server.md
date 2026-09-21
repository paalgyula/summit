# WebSocket World Server + Web Client

## Overview

Add WebSocket transport to the world server enabling a full WebGL-based WoW 3.3.5a game client in the browser. This is a long-term, multi-phase project.

**Scope**: Full game client (WebGL/WebGPU)
**Auth**: JWT/API token (skips SRP6)
**Transport**: Parallel WebSocket listener alongside existing TCP

---

## Phase 1: WebSocket Transport Adapter

**Goal**: WebSocket clients can connect, authenticate via JWT, and exchange raw WoW binary packets through the existing handler pipeline.

### 1.1 Add `gorilla/websocket` dependency

```bash
go get github.com/gorilla/websocket
```

### 1.2 Create `pkg/summit/world/websocket/` package

**New file: `pkg/summit/world/websocket/conn.go`**

Create a `WebSocketConn` adapter implementing `io.ReadWriteCloser`:
- Wraps `*websocket.Conn`
- `Read()` — reads a binary WebSocket message (one complete WoW packet: 6-byte header + payload)
- `Write()` — writes a binary WebSocket message (complete framed packet)
- `Close()` — sends close frame, closes underlying connection
- Each WebSocket binary frame = one complete WoW packet (no chunking, no message batching)
- No RC4 encryption — TLS provides transport security

**Key design**: The existing `protocol.WoWSocket` takes `io.ReadWriteCloser`. This adapter lets WebSocket connections flow through the exact same packet pipeline.

### 1.3 JWT Auth flow

**New file: `pkg/summit/world/websocket/auth.go`**

WebSocket clients authenticate differently from TCP:
1. Client connects to WebSocket endpoint with JWT in query param or first message
2. Server validates JWT (signed with a shared secret or RSA key from config)
3. JWT payload contains: `account_id`, `session_key` (40-byte WoW session key), `exp`
4. Server creates `WorldSession` with the session key, skips `SMSG_AUTH_CHALLENGE`/`CMSG_AUTH_SESSION` handshake
5. RC4 encryption is skipped — `WowCrypt` not initialized for WS clients
6. Client is now in the same state as a post-auth TCP client

### 1.4 Server option for WebSocket listener

**Modify: `pkg/summit/world/server_options.go`**

Add:
```go
func WithWebSocket(addr string) ServerOption {
    return func(s *Server) error {
        s.wsAddr = addr
        return nil
    }
}
```

### 1.5 WebSocket listener in Server

**Modify: `pkg/summit/world/server.go`**

Add `startWebSocketListener()` alongside existing `startListener()`:
- Listen on configured address (e.g., `:8130`)
- HTTP upgrade handler using `gorilla/websocket.Upgrader`
- On new connection: validate JWT, create `WorldSession`, run `handleConnection()`
- Integrate into `Start()` method

### 1.6 Skip RC4 for WebSocket sessions

**Modify: `pkg/summit/world/worldsession.go`**

Add a flag or option to `WorldSession` indicating WebSocket transport:
- When set, `WowCrypt` is not initialized
- `connectionWriter` sends packets unencrypted
- `connectionReader` expects unencrypted packets

**Alternative**: Modify `protocol.WoWSocket` to accept a `noCrypt` option:
```go
func NewWoWSocket(conn io.ReadWriteCloser, opts ...WoWSocketOption) *WoWSocket
type WoWSocketOption func(*WoWSocket)
func WithoutEncryption() WoWSocketOption
```

### 1.7 Configuration

**Modify: `cmd/summit/summit.go`** and `summit.yaml`

Add config section:
```yaml
websocket:
  enabled: true
  address: ":8130"
  jwt_secret: "your-secret-key"
  # or
  jwt_public_key_path: "/path/to/public.pem"
```

### 1.8 Tests

- Unit test: `WebSocketConn` read/write round-trip
- Integration test: WebSocket client performs JWT auth → sends CMSG_PING → receives SMSG_PONG
- Regression test: existing TCP clients still connect and authenticate

---

## Phase 2: Web Client Foundation

**Goal**: Minimal browser client that connects, authenticates, and renders a 3D character in an empty world.

### 2.1 Project setup

**New directory: `web/`** (or separate repo)

- TypeScript + Vite build system
- Three.js or Babylon.js for WebGL rendering
- `WebSocket` API for transport

### 2.2 WebSocket transport layer

```typescript
class WoWWebSocket {
  private ws: WebSocket;
  
  connect(url: string, jwt: string) {
    this.ws = new WebSocket(`${url}?token=${jwt}`);
    this.ws.binaryType = 'arraybuffer';
  }
  
  sendPacket(opcode: number, payload: ArrayBuffer) {
    // Frame: [2B length BE][4B opcode LE][payload]
    const header = new ArrayBuffer(6);
    const view = new DataView(header);
    view.setUint16(0, payload.byteLength + 4, false); // BE
    view.setUint32(2, opcode, true); // LE
    this.ws.send(mergeBuffers(header, payload));
  }
  
  onPacket(callback: (opcode: number, data: ArrayBuffer) => void) {
    this.ws.onmessage = (e) => {
      const view = new DataView(e.data);
      const length = view.getUint16(0, false);
      const opcode = view.getUint32(2, true);
      const payload = e.data.slice(6);
      callback(opcode, payload);
    };
  }
}
```

### 2.3 Opcode definitions

Port opcode constants from generated Go code to TypeScript:
- Parse `pkg/wow/protocol/opcodes_generated.go` or use opcode definitions
- Generate TypeScript enum from same source

### 2.4 Packet parser/builder

Binary packet serialization matching Go server's format:
- Little-endian integers
- Packed GUIDs (WoW's variable-length GUID encoding)
- String encoding (null-terminated, UTF-8)
- C-style struct packing

### 2.5 Auth flow in browser

1. User logs in via HTTP API (separate auth endpoint or existing auth server)
2. Receives JWT with session key
3. Opens WebSocket with JWT
4. Server creates WorldSession, sends initial packets
5. Client enters character list → world

### 2.6 Minimal rendering

- Load character model (M2 format)
- Load terrain (ADT format) — initially flat ground
- Basic camera controls (WASD + mouse)
- Character movement (CMSG_MOVEMENT)

---

## Phase 3: Core Game Systems

**Goal**: Functional client with character movement, NPC interaction, and combat.

### 3.1 Object spawning

Handle server packets:
- `SMSG_UPDATE_OBJECT` — spawn/despawn objects, movement updates
- `SMSGCreatureQueryResponse` / `SMSGItemQueryResponse` — entity metadata
- Object GUID → entity mapping

### 3.2 Movement system

- Client sends `CMSG_MOVE_*` packets (start/stop/turn)
- Server validates and broadcasts via `SMSG_MONSTER_MOVE`
- Interpolate movement between server updates

### 3.3 Chat system

- `SMSG_MESSAGECHAT` — receive messages
- `CMSG_MESSAGECHAT` — send messages
- Chat window UI overlay (HTML/CSS)

### 3.4 Combat basics

- Target selection (`CMSG_SETTARGET`)
- Spell casting (`CMSG_CAST_SPELL`)
- Damage/healing numbers (`SMSG_SPELLNONMELEEDAMAGELOG`)

---

## Phase 4: Advanced Systems

**Goal**: Full gameplay parity.

### 4.1 Inventory & equipment
### 4.2 Quest log
### 4.3 Spells & talents
### 4.4 NPCs & quests
### 4.5 Dungeon/raid instancing
### 4.6 Auction house
### 4.7 Social features (friends, guild)

---

## Technical Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| WebSocket library | `gorilla/websocket` | Battle-tested, widely used in Go |
| Packet framing | Binary frames, 1:1 with WoW packets | Minimal overhead, direct translation |
| RC4 encryption | Skipped for WS clients | TLS provides transport security |
| Auth method | JWT with session key | Simple, stateless, no SRP6 complexity |
| Web framework | Three.js | Mature, good WoW model support via community libs |
| Build system | Vite + TypeScript | Fast dev, good tooling |

## File Changes Summary

| File | Action | Description |
|------|--------|-------------|
| `pkg/summit/world/websocket/conn.go` | NEW | WebSocket↔io.ReadWriteCloser adapter |
| `pkg/summit/world/websocket/auth.go` | NEW | JWT validation + session creation |
| `pkg/summit/world/websocket/handler.go` | NEW | HTTP upgrade handler |
| `pkg/summit/world/server.go` | MODIFY | Add WebSocket listener |
| `pkg/summit/world/server_options.go` | MODIFY | Add `WithWebSocket()` option |
| `pkg/summit/world/worldsession.go` | MODIFY | Add no-encrypt mode |
| `pkg/wow/protocol/wowsocket.go` | MODIFY | Add `WithoutEncryption()` option |
| `cmd/summit/summit.go` | MODIFY | Wire WebSocket config |
| `summit.yaml` | MODIFY | Add websocket config section |
| `web/` | NEW | Entire web client project |
