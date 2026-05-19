// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

package wg_test

import (
	"testing"

	"github.com/PharosVPN/helm/internal/wg"
)

func TestObfuscationIsZero(t *testing.T) {
	if !(wg.Obfuscation{}).IsZero() {
		t.Error("empty Obfuscation should be zero")
	}
	// Junk sizes alone, with no magic headers, still count as unconfigured —
	// a real node always sets H1-H4.
	if !(wg.Obfuscation{Jc: 4, S1: 30}).IsZero() {
		t.Error("Obfuscation without magic headers should be zero")
	}
	configured := wg.Obfuscation{H1: 1515448789, H2: 2406647629, H3: 3604601557, H4: 1124628755}
	if configured.IsZero() {
		t.Error("Obfuscation with magic headers should not be zero")
	}
}
