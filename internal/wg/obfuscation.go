// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

package wg

// Obfuscation is one node's AmneziaWG obfuscation parameter set (DESIGN §3).
// Every node runs AmneziaWG — there is no plain-WireGuard path — and each node
// randomises its own values for traffic diversity, so the set is per-node, not
// fleet-wide. buoy generates and applies it; helm stores it and a client must
// receive the exact values to build a tunnel that handshakes.
//
// Field names match the AmneziaWG config keys verbatim:
//
//   - Jc/Jmin/Jmax: junk-packet count and size range
//   - S1-S4: junk sizes for init / response / cookie-reply / transport packets
//   - H1-H4: magic header values for those four packet types
//   - I1-I5: special-junk packet templates injected at handshake stages
type Obfuscation struct {
	Jc   uint32 `json:"jc"`
	Jmin uint32 `json:"jmin"`
	Jmax uint32 `json:"jmax"`
	S1   uint32 `json:"s1"`
	S2   uint32 `json:"s2"`
	S3   uint32 `json:"s3"`
	S4   uint32 `json:"s4"`
	H1   uint32 `json:"h1"`
	H2   uint32 `json:"h2"`
	H3   uint32 `json:"h3"`
	H4   uint32 `json:"h4"`
	I1   string `json:"i1,omitempty"`
	I2   string `json:"i2,omitempty"`
	I3   string `json:"i3,omitempty"`
	I4   string `json:"i4,omitempty"`
	I5   string `json:"i5,omitempty"`
}

// IsZero reports whether o carries no obfuscation parameters — the state of a
// node that has not yet reported its AmneziaWG configuration. The magic
// headers H1-H4 are always non-zero on a configured node (AmneziaWG reserves
// the low values), so they are the reliable signal.
func (o Obfuscation) IsZero() bool {
	return o.H1 == 0 && o.H2 == 0 && o.H3 == 0 && o.H4 == 0
}
