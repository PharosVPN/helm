# helm — Build Brief

**Read first:** `docs/BUILD.md`, then `docs/DESIGN.md`. This brief assumes both.

`helm` is the **core** project — built collaboratively (operator + Claude), not
delegated to a subagent. This file is the roadmap, not a hand-off spec.

---

## Scope

`helm` is the controller. It owns: fleet state, the CA, cloud provisioning, the
outbound control loop to `buoy` nodes, the embedded `beacon` relay + remote-relay
dialer, the account/sync service, and the admin Web UI.

It does **not** run a data plane and opens **no inbound ports**.

## Prior art to port

The current `amnezia-travelvpn` repo (`/Users/khalefa/Projects/amnezia-travelvpn`)
is a working single-controller version. Port forward, do not copy blindly:

- **Keep:** SQLite + Goose migrations, YAML snapshot projections, JWT + Argon2id
  auth, the koanf config loader, the AmneziaWG/XRay profile model, the
  embedded-SvelteKit pattern.
- **Replace:** SSH-to-node *control* and shell config-editing → outbound
  mTLS/gRPC to `buoy`. SSH is kept, but only as the agent install/update
  channel — never for control.
- **Add:** the CA/PKI, account system (users as auth principals), E2E profile
  encryption, live event streaming + WebSocket, optimistic concurrency, the
  embedded `beacon`, `.pharos` export.

## Milestones

| # | Milestone | Output |
|---|---|---|
| M1 | Skeleton | `cmd/helm`, config, SQLite + migrations, CA generation on first run, `helm init --personal/--enterprise` |
| M2 | Node onboarding | SSH agent install/update; `helm nodes add` — CSR signing, host-key pinning, `helm ssh-key` |
| M3 | Control loop | gRPC client to `buoy`; push config, push/revoke peers; status/metrics |
| M4 | Live plane | consume `buoy` event stream; WebSocket fan-out to admin browsers |
| M5 | Admin UI | SvelteKit SPA, auth, fleet/users/peers/admins screens, optimistic concurrency (409 on stale write) |
| M6 | Accounts & sync | users as auth principals + roles; E2E profile encryption (§8); embedded `beacon`; remote-`beacon` reverse-tunnel dialer |
| M7 | Distribution | `.pharos` export (all `enc` modes), enrollment-ticket QR, Amnezia-compat `.vpn` export |

## Contracts `helm` owns

`helm` defines the wire contracts the other repos consume. As each milestone
lands, update `docs/proto/`:

- `buoy` control service (M3) + event stream (M4)
- `beacon` reverse-tunnel + relayed client service (M6)
- `caravel` account/sync service (M6/M7)

Coordinate every proto change with the dependent subproject's BUILD.

### Pinned `beacon` ↔ `helm` identifiers

Confirmed with the `beacon` agent during its R1. The M6b relayed-client proto
and PKI **must** use these exact values:

| Role | Value |
|---|---|
| Injected verified-device-fingerprint metadata key | `x-pharos-device-fp` |
| Stripped client-metadata prefix (anti-spoofing) | `x-pharos-` |
| Backend delegation cert `Organization` | `PharosVPN Relay` |
| `helm` gRPC-leg leaf `CN` / backend SNI | `helm-grpc` |

## Non-negotiables

- No inbound ports. All node/relay connections are helm-initiated.
- The CA private key never leaves `helm`'s SQLite.
- User private keys exist on `helm` only as passphrase-wrapped blobs (DESIGN §8).
- Every mutable table row has `version` + `updated_at` (DESIGN §7).
