// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

// Package provision places a user's device onto the fleet: it allocates the
// device a tunnel address and a peer (keypair + preshared key) on every
// ready node, then issues a sealed profile (DESIGN §8, §9).
package provision

import (
	"context"
	"database/sql"
	"fmt"
	"net"

	"github.com/PharosVPN/helm/internal/account"
	"github.com/PharosVPN/helm/internal/fleet"
	"github.com/PharosVPN/helm/internal/profile"
	"github.com/PharosVPN/helm/internal/wg"
)

// awgPort is the UDP port a buoy serves AmneziaWG on (DESIGN §2).
const awgPort = "443"

// Result reports what ProvisionDevice produced.
type Result struct {
	Device         account.Device
	TunnelIP       string
	PeerCount      int
	ProfileVersion int64
}

// ProvisionDevice gives a device a tunnel address and an AmneziaWG peer on
// every ready node, records the peers, and issues a freshly sealed profile to
// the device's owner. A node is "ready" once it has reported its WG public
// key and has a public address.
//
// The peer records are helm's desired state; pushing them to buoy over the
// control channel is the control loop's job.
func ProvisionDevice(ctx context.Context, db *sql.DB, deviceID, vpnSubnet string) (Result, error) {
	device, err := account.GetDevice(ctx, db, deviceID)
	if err != nil {
		return Result{}, err
	}

	keys, err := wg.GenerateKeyPair()
	if err != nil {
		return Result{}, err
	}
	tunnelIP, err := fleet.AllocateDeviceIP(ctx, db, vpnSubnet)
	if err != nil {
		return Result{}, err
	}
	nodes, err := fleet.ListNodes(ctx, db)
	if err != nil {
		return Result{}, err
	}

	var buildNodes []profile.BuildNode
	for _, n := range nodes {
		if n.WGPublicKey == "" || n.PublicIP == "" {
			continue // node has not reported its data-plane key yet
		}
		psk := profile.GeneratePresharedKey()
		if _, err := fleet.CreatePeer(ctx, db, fleet.Peer{
			NodeID:       n.ID,
			DeviceID:     device.ID,
			Protocol:     profile.ProtocolAmneziaWG,
			PublicKey:    keys.PublicKey,
			AllowedIP:    tunnelIP,
			PresharedKey: psk,
		}); err != nil {
			return Result{}, fmt.Errorf("provision: peer on %s: %w", n.ID, err)
		}
		buildNodes = append(buildNodes, profile.BuildNode{
			ID:           n.ID,
			Name:         n.Name,
			Region:       n.Region,
			Endpoint:     net.JoinHostPort(n.PublicIP, awgPort),
			WGPublicKey:  n.WGPublicKey,
			PresharedKey: psk,
			AllowedIPs:   []string{"0.0.0.0/0", "::/0"},
		})
	}

	prof := profile.Build(profile.BuildInput{
		User:        device.UserID,
		DeviceWGKey: keys.PrivateKey,
		TunnelIP:    tunnelIP,
		Nodes:       buildNodes,
	})
	revision, err := profile.Issue(ctx, db, device.UserID, prof)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Device:         device,
		TunnelIP:       tunnelIP,
		PeerCount:      len(buildNodes),
		ProfileVersion: revision,
	}, nil
}
