// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

package profile

import "encoding/json"

// ProtocolVersionAmneziaWG is the version tag for amneziawg protocol blocks.
const ProtocolVersionAmneziaWG = 2

// BuildNode is one node's contribution to a profile — the node identity plus
// the device's peer material on it.
type BuildNode struct {
	ID          string
	Name        string
	Region      string
	Endpoint    string // host:port the client dials
	WGPublicKey string // the node's AmneziaWG server public key
	// PresharedKey is the per-(device,node) 256-bit PSK (decision 15).
	PresharedKey string
	AllowedIPs   []string
}

// BuildInput is everything needed to assemble a device's profile.
type BuildInput struct {
	User        string
	FleetID     string
	DeviceWGKey string // the device's AmneziaWG private key
	TunnelIP    string // the device's allocated VPN address
	Nodes       []BuildNode
}

// amneziaWGParams is the params block of an amneziawg protocol entry.
type amneziaWGParams struct {
	PrivateKey   string   `json:"private_key"`
	Address      string   `json:"address"`
	PublicKey    string   `json:"public_key"`
	PresharedKey string   `json:"preshared_key"`
	Endpoint     string   `json:"endpoint"`
	AllowedIPs   []string `json:"allowed_ips"`
}

// Build assembles a populated Profile from a device's peers. Revision and
// timestamps are filled in by Issue when the profile is sealed.
func Build(in BuildInput) Profile {
	p := Profile{FleetID: in.FleetID, User: in.User}
	for _, n := range in.Nodes {
		params, _ := json.Marshal(amneziaWGParams{
			PrivateKey:   in.DeviceWGKey,
			Address:      in.TunnelIP + "/32",
			PublicKey:    n.WGPublicKey,
			PresharedKey: n.PresharedKey,
			Endpoint:     n.Endpoint,
			AllowedIPs:   n.AllowedIPs,
		})
		p.Nodes = append(p.Nodes, Node{
			ID:        n.ID,
			Name:      n.Name,
			Region:    n.Region,
			Endpoints: []string{n.Endpoint},
			Protocols: []Protocol{{
				Type:   ProtocolAmneziaWG,
				V:      ProtocolVersionAmneziaWG,
				Params: params,
			}},
		})
	}
	return p
}
