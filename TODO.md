# MCP Server - Code Review Issues


## P1 - Critical

- [ ] **StopAgents doesn't terminate processes** — `internal/mcp/handlers.go:302-323`

  The MCP `stop_agents` handler only calls `agent.ClearState()` without killing actual agent processes. Agents continue running as orphaned processes while the system reports success.

  **Fix:** Port the process termination logic from `internal/cmd/stop.go:81-108` (uses `lsof` + `kill -TERM`).

- [ ] **ClearState wipes ALL agents** — `internal/mcp/handlers.go:320`

  When stopping a single agent by name, `agent.ClearState()` is called unconditionally, wiping state for ALL agents instead of just the specified one.

  **Fix:** Remove only the specified agent from state, or call a selective clear function.

## P2 - Important

- [ ] **Missing path sandboxing** — `internal/mcp/handlers.go:67-74`

  `PRDPath` accepts any absolute path without validating it's within an allowed directory. Risk of reading/writing files anywhere on the filesystem.

  **Fix:** Validate that `absPath` has the project root or working directory as a prefix.

- [ ] **Hardcoded localhost in OAuth** — `internal/mcp/server.go:171`

  OAuth `ServerURL` is hardcoded to `http://localhost:%d`. Breaks OAuth flows when accessed via network IP or domain.

  **Fix:** Make `ServerURL` configurable or derive from request host.

## P3 - Minor

- [ ] **TotalDuration never populated** — `internal/mcp/handlers.go:213-218`

  `RunPipelineOutput.TotalDuration` field exists in `types.go:39` but is never set. MCP clients always see empty duration.

  **Fix:** Track start time, compute duration, populate field before returning.

- [ ] **Stale state detection** — `internal/mcp/handlers.go` (GetStatus)

  `GetStatus` trusts the state file ports without verifying processes are actually running. May report incorrect status after crashes.

  **Fix:** Verify process is alive (health check or port probe) before reporting status.

---

## Related Files

| File | Purpose |
|------|---------|
| `internal/mcp/handlers.go` | MCP tool business logic |
| `internal/mcp/server.go` | Server + transport methods |
| `internal/mcp/types.go` | Input/output types |
| `internal/cmd/stop.go` | Reference implementation for stopping agents |
