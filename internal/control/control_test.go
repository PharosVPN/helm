// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

package control_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"net"
	"path/filepath"
	"testing"

	"github.com/PharosVPN/helm/internal/control"
	"github.com/PharosVPN/helm/internal/db"
	buoyv1 "github.com/PharosVPN/helm/internal/gen/pharos/buoy/v1"
	"github.com/PharosVPN/helm/internal/pki"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// fakeNode is a minimal NodeControl server for exercising the control client.
type fakeNode struct {
	buoyv1.UnimplementedNodeControlServer
}

func (fakeNode) GetStatus(context.Context, *buoyv1.GetStatusRequest) (*buoyv1.GetStatusResponse, error) {
	return &buoyv1.GetStatusResponse{AgentVersion: "buoy 9.9.9", UptimeSeconds: 42}, nil
}

func (fakeNode) AddPeer(_ context.Context, req *buoyv1.AddPeerRequest) (*buoyv1.PeerResponse, error) {
	return &buoyv1.PeerResponse{PeerId: req.GetPeer().GetId(), Applied: true}, nil
}

// TestControlClientOverMTLS dials a fake buoy server over the full mTLS path:
// helm's controller cert vs a Fleet-CA node cert, both chaining to the root.
func TestControlClientOverMTLS(t *testing.T) {
	ctx := context.Background()

	conn, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}
	bundle, _, err := pki.EnsureCA(ctx, conn)
	if err != nil {
		t.Fatalf("EnsureCA: %v", err)
	}
	controllerCert, _, err := pki.EnsureControllerCert(ctx, conn, bundle.Fleet)
	if err != nil {
		t.Fatalf("EnsureControllerCert: %v", err)
	}

	// Node server certificate, signed off the Fleet CA with a localhost SAN.
	nodeKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("node key: %v", err)
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader,
		&x509.CertificateRequest{Subject: pkix.Name{CommonName: "buoy-test"}}, nodeKey)
	if err != nil {
		t.Fatalf("node CSR: %v", err)
	}
	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})
	signed, err := pki.SignNodeCSR(bundle.Fleet, csrPEM, []net.IP{net.ParseIP("127.0.0.1")}, nil)
	if err != nil {
		t.Fatalf("SignNodeCSR: %v", err)
	}
	nodeKeyDER, err := x509.MarshalPKCS8PrivateKey(nodeKey)
	if err != nil {
		t.Fatalf("marshal node key: %v", err)
	}
	nodeKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: nodeKeyDER})
	serverCert, err := tls.X509KeyPair(concat(signed.CertPEM, bundle.Fleet.CertPEM), nodeKeyPEM)
	if err != nil {
		t.Fatalf("server keypair: %v", err)
	}

	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(bundle.Root.CertPEM)

	srv := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    roots,
		MinVersion:   tls.VersionTLS13,
	})))
	buoyv1.RegisterNodeControlServer(srv, fakeNode{})

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go srv.Serve(lis) //nolint:errcheck // returns on Stop
	defer srv.Stop()

	dialer, err := control.NewDialer(
		concat(controllerCert.CertPEM, bundle.Fleet.CertPEM),
		controllerCert.KeyPEM,
		bundle.Root.CertPEM)
	if err != nil {
		t.Fatalf("NewDialer: %v", err)
	}
	client, err := dialer.Dial(lis.Addr().String())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	status, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.GetAgentVersion() != "buoy 9.9.9" {
		t.Errorf("agent version: got %q", status.GetAgentVersion())
	}

	peerResp, err := client.AddPeer(ctx, &buoyv1.Peer{
		Id:        "peer-1",
		Protocol:  buoyv1.Protocol_PROTOCOL_AMNEZIAWG,
		PublicKey: "dGVzdA==",
	})
	if err != nil {
		t.Fatalf("AddPeer: %v", err)
	}
	if !peerResp.GetApplied() || peerResp.GetPeerId() != "peer-1" {
		t.Errorf("AddPeer: got %+v", peerResp)
	}
}

func TestNewDialerRejectsBadInput(t *testing.T) {
	if _, err := control.NewDialer([]byte("nope"), []byte("nope"), []byte("nope")); err == nil {
		t.Error("expected NewDialer to reject malformed PEM")
	}
}

// concat joins PEM blocks without aliasing the inputs.
func concat(a, b []byte) []byte {
	out := make([]byte, 0, len(a)+len(b))
	out = append(out, a...)
	return append(out, b...)
}
