// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

package config

import "fmt"

// Preset returns the starting Config for a deployment posture. These encode the
// "personal vs enterprise" defaults table in DESIGN §12. `helm init` writes the
// result to disk; the operator edits it from there.
func Preset(p Posture) (Config, error) {
	switch p {
	case PosturePersonal:
		return personalPreset(), nil
	case PostureEnterprise:
		return enterprisePreset(), nil
	default:
		return Config{}, fmt.Errorf("unknown posture %q", p)
	}
}

// common returns the fields shared by every posture.
func common() Config {
	return Config{
		StateDir: "./state",
		Log:      LogConfig{Level: "info"},
		UI:       UIConfig{Listen: "127.0.0.1:8443"},
		Beacon:   BeaconConfig{Embedded: true},
		Accounts: AccountsConfig{Sync: true},
		Reality:  RealityConfig{DecoySite: "www.microsoft.com"},
		// BuoyBinaryURL is left for the operator to point at a buoy release.
		Node: NodeConfig{SSHUser: "root", SSHPort: 22},
	}
}

func personalPreset() Config {
	c := common()
	c.Posture = PosturePersonal
	c.Protocols = ProtocolsConfig{AmneziaWG: true, XRay: false}
	c.Beacon.Remote = false
	c.Retention = RetentionConfig{AuditDays: 30, MetricsDays: 7}
	c.Fleet = FleetConfig{Regions: []string{}, IdleNodes: false}
	return c
}

func enterprisePreset() Config {
	c := common()
	c.Posture = PostureEnterprise
	c.Protocols = ProtocolsConfig{AmneziaWG: true, XRay: true}
	c.Beacon.Remote = true
	c.Retention = RetentionConfig{AuditDays: 365, MetricsDays: 90}
	c.Fleet = FleetConfig{Regions: []string{}, IdleNodes: true}
	return c
}
