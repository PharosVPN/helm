// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

package pki_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"net"
	"path/filepath"
	"testing"

	"github.com/PharosVPN/helm/internal/db"
	"github.com/PharosVPN/helm/internal/pki"
)

// makeCSR generates a node keypair and a PEM-encoded CSR, as buoy would on the
// node. The private key never leaves; only csrPEM is returned.
func makeCSR(t *testing.T, cn string) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	der, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: cn},
		DNSNames: []string{cn + ".fleet.internal"},
	}, key)
	if err != nil {
		t.Fatalf("CreateCertificateRequest: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der})
}

func TestSignNodeCSRChains(t *testing.T) {
	b, err := pki.GenerateBundle()
	if err != nil {
		t.Fatalf("GenerateBundle: %v", err)
	}

	signed, err := pki.SignNodeCSR(b.Fleet, makeCSR(t, "buoy-ams-1"),
		[]net.IP{net.ParseIP("203.0.113.7")}, nil)
	if err != nil {
		t.Fatalf("SignNodeCSR: %v", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(b.Root.Cert)
	inter := x509.NewCertPool()
	inter.AddCert(b.Fleet.Cert)
	if _, err := signed.Cert.Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: inter,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}); err != nil {
		t.Fatalf("signed cert does not chain root->fleet->leaf: %v", err)
	}

	if len(signed.Cert.IPAddresses) != 1 || !signed.Cert.IPAddresses[0].Equal(net.ParseIP("203.0.113.7")) {
		t.Errorf("extra IP SAN missing: %v", signed.Cert.IPAddresses)
	}
	if len(signed.Cert.DNSNames) == 0 {
		t.Error("CSR DNS SAN not carried onto the certificate")
	}
}

func TestSignNodeCSRRejectsNonFleetCA(t *testing.T) {
	b, err := pki.GenerateBundle()
	if err != nil {
		t.Fatalf("GenerateBundle: %v", err)
	}
	csr := makeCSR(t, "x")
	if _, err := pki.SignNodeCSR(b.Root, csr, nil, nil); err == nil {
		t.Error("expected SignNodeCSR to reject the root CA")
	}
	if _, err := pki.SignNodeCSR(b.Device, csr, nil, nil); err == nil {
		t.Error("expected SignNodeCSR to reject the device CA")
	}
}

func TestSignNodeCSRRejectsGarbage(t *testing.T) {
	b, err := pki.GenerateBundle()
	if err != nil {
		t.Fatalf("GenerateBundle: %v", err)
	}
	if _, err := pki.SignNodeCSR(b.Fleet, []byte("not a pem"), nil, nil); err == nil {
		t.Error("expected SignNodeCSR to reject non-PEM input")
	}
}

func TestRecordNodeCert(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	ctx := context.Background()

	if _, err := conn.ExecContext(ctx,
		`INSERT INTO nodes (id, name, region) VALUES ('nod_test', 'ams-1', 'eu')`); err != nil {
		t.Fatalf("seed node: %v", err)
	}

	b, err := pki.GenerateBundle()
	if err != nil {
		t.Fatalf("GenerateBundle: %v", err)
	}
	signed, err := pki.SignNodeCSR(b.Fleet, makeCSR(t, "buoy-ams-1"), nil, nil)
	if err != nil {
		t.Fatalf("SignNodeCSR: %v", err)
	}

	id, err := pki.RecordNodeCert(ctx, conn, "nod_test", signed)
	if err != nil {
		t.Fatalf("RecordNodeCert: %v", err)
	}

	var serial string
	if err := conn.QueryRowContext(ctx,
		`SELECT serial FROM node_certs WHERE id = ?`, id).Scan(&serial); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if serial != signed.Serial {
		t.Errorf("serial: got %q want %q", serial, signed.Serial)
	}
}
