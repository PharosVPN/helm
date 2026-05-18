// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

package provision_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PharosVPN/helm/internal/account"
	"github.com/PharosVPN/helm/internal/db"
	"github.com/PharosVPN/helm/internal/e2e"
	"github.com/PharosVPN/helm/internal/fleet"
	"github.com/PharosVPN/helm/internal/profile"
	"github.com/PharosVPN/helm/internal/provision"
)

const subnet = "10.86.0.0/16"

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}
	return conn
}

// enrolledUser creates a user with an E2E encryption key and returns the user
// ID and the keypair (for decrypting the issued profile).
func enrolledUser(t *testing.T, conn *sql.DB) (string, e2e.KeyPair) {
	t.Helper()
	ctx := context.Background()
	u, err := account.CreateUser(ctx, conn, account.User{Email: "u@example.com"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	kp, err := e2e.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	wrapped, err := e2e.WrapPrivateKey("pw", kp.Private)
	if err != nil {
		t.Fatalf("WrapPrivateKey: %v", err)
	}
	if err := account.SetEncryptionKey(ctx, conn, u.ID, kp.Public, wrapped); err != nil {
		t.Fatalf("SetEncryptionKey: %v", err)
	}
	return u.ID, kp
}

func TestProvisionDevice(t *testing.T) {
	conn := newDB(t)
	ctx := context.Background()
	userID, kp := enrolledUser(t, conn)

	device, err := account.CreateDevice(ctx, conn, account.Device{UserID: userID, Name: "phone"})
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	// Two ready nodes and one still pending (no WG key) — the pending one
	// must be skipped.
	for _, n := range []fleet.Node{
		{Name: "ams-1", Region: "eu", PublicIP: "203.0.113.7", WGPublicKey: "bm9kZS1hbXMtd2cta2V5LWJhc2U2NA=="},
		{Name: "fra-1", Region: "eu", PublicIP: "203.0.113.8", WGPublicKey: "bm9kZS1mcmEtd2cta2V5LWJhc2U2NA=="},
		{Name: "pending", Region: "us"},
	} {
		if _, err := fleet.CreateNode(ctx, conn, n); err != nil {
			t.Fatalf("CreateNode %s: %v", n.Name, err)
		}
	}

	res, err := provision.ProvisionDevice(ctx, conn, device.ID, subnet)
	if err != nil {
		t.Fatalf("ProvisionDevice: %v", err)
	}
	if res.PeerCount != 2 {
		t.Errorf("peer count: got %d want 2 (pending node should be skipped)", res.PeerCount)
	}
	if !strings.HasPrefix(res.TunnelIP, "10.86.") {
		t.Errorf("tunnel IP %q not in subnet", res.TunnelIP)
	}

	peers, err := fleet.ListPeersByDevice(ctx, conn, device.ID)
	if err != nil {
		t.Fatalf("ListPeersByDevice: %v", err)
	}
	if len(peers) != 2 {
		t.Fatalf("recorded peers: got %d want 2", len(peers))
	}
	for _, p := range peers {
		if p.PresharedKey == "" || p.PublicKey == "" || p.AllowedIP != res.TunnelIP {
			t.Errorf("peer not fully provisioned: %+v", p)
		}
	}

	// The issued profile decrypts to a populated profile.
	ciphertext, _, err := profile.LatestCiphertext(ctx, conn, userID)
	if err != nil {
		t.Fatalf("LatestCiphertext: %v", err)
	}
	signing, _, err := profile.EnsureSigningKey(ctx, conn)
	if err != nil {
		t.Fatalf("EnsureSigningKey: %v", err)
	}
	var bundle e2e.SealedBundle
	if err := json.Unmarshal(ciphertext, &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	plaintext, err := e2e.Open(bundle, kp.Private, signing.Public)
	if err != nil {
		t.Fatalf("e2e.Open: %v", err)
	}
	var prof profile.Profile
	if err := json.Unmarshal(plaintext, &prof); err != nil {
		t.Fatalf("unmarshal profile: %v", err)
	}
	if len(prof.Nodes) != 2 {
		t.Fatalf("profile nodes: got %d want 2", len(prof.Nodes))
	}
	proto := prof.Nodes[0].Protocols[0]
	if proto.Type != profile.ProtocolAmneziaWG {
		t.Errorf("protocol type: got %q", proto.Type)
	}
	// The amneziawg params must carry the device's private key.
	var params struct {
		PrivateKey string `json:"private_key"`
		PublicKey  string `json:"public_key"`
	}
	if err := json.Unmarshal(proto.Params, &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if params.PrivateKey == "" || params.PublicKey == "" {
		t.Errorf("amneziawg params incomplete: %+v", params)
	}
}

func TestAllocateDeviceIPSequential(t *testing.T) {
	conn := newDB(t)
	ctx := context.Background()
	userID, _ := enrolledUser(t, conn)

	if _, err := fleet.CreateNode(ctx, conn, fleet.Node{
		Name: "n1", Region: "eu", PublicIP: "203.0.113.7", WGPublicKey: "d2cta2V5LWZvci10aGUtdGVzdC1ub2Rl",
	}); err != nil {
		t.Fatalf("CreateNode: %v", err)
	}

	var ips []string
	for i := 0; i < 3; i++ {
		d, err := account.CreateDevice(ctx, conn, account.Device{UserID: userID, Name: "d"})
		if err != nil {
			t.Fatalf("CreateDevice: %v", err)
		}
		res, err := provision.ProvisionDevice(ctx, conn, d.ID, subnet)
		if err != nil {
			t.Fatalf("ProvisionDevice: %v", err)
		}
		ips = append(ips, res.TunnelIP)
	}
	// Distinct, ascending allocations.
	seen := map[string]bool{}
	for _, ip := range ips {
		if seen[ip] {
			t.Fatalf("duplicate tunnel IP %q in %v", ip, ips)
		}
		seen[ip] = true
	}
}
