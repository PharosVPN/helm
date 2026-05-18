// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

// Package config defines helm's configuration model, the personal/enterprise
// presets, and the koanf-based loader.
package config

// Posture is the deployment posture chosen at `helm init`.
type Posture string

const (
	// PosturePersonal — one operator, a handful of nodes.
	PosturePersonal Posture = "personal"
	// PostureEnterprise — a team managing many users across many regions.
	PostureEnterprise Posture = "enterprise"
)

// Config is the full helm configuration. It is persisted as YAML and reloaded
// on every start. Field tags are shared between koanf (load) and yaml (write).
type Config struct {
	// Posture records which preset this deployment was initialised from.
	Posture Posture `koanf:"posture" yaml:"posture"`
	// StateDir holds the SQLite database, snapshots, and other on-disk state.
	StateDir string `koanf:"state_dir" yaml:"state_dir"`

	Log       LogConfig       `koanf:"log" yaml:"log"`
	UI        UIConfig        `koanf:"ui" yaml:"ui"`
	Protocols ProtocolsConfig `koanf:"protocols" yaml:"protocols"`
	Beacon    BeaconConfig    `koanf:"beacon" yaml:"beacon"`
	Accounts  AccountsConfig  `koanf:"accounts" yaml:"accounts"`
	Retention RetentionConfig `koanf:"retention" yaml:"retention"`
	Reality   RealityConfig   `koanf:"reality" yaml:"reality"`
	Fleet     FleetConfig     `koanf:"fleet" yaml:"fleet"`
	Node      NodeConfig      `koanf:"node" yaml:"node"`
	Admin     AdminConfig     `koanf:"admin" yaml:"admin"`
}

// AdminConfig holds the fixed controller-admin account (DESIGN §8). The
// password here is the source of truth — helm re-syncs it into the database
// on every start, so editing it and restarting changes the admin login.
type AdminConfig struct {
	// Password is the fixed admin's login password. `helm init` generates a
	// strong random value; the operator may replace it and restart.
	Password string `koanf:"password" yaml:"password"`
}

// NodeConfig holds defaults for SSH-based node onboarding (DESIGN §5). helm
// reaches a node over SSH only to install and update the buoy agent.
type NodeConfig struct {
	// BuoyBinaryURL is the default download URL for the buoy agent, used by
	// `helm nodes add` when no local binary is supplied.
	BuoyBinaryURL string `koanf:"buoy_binary_url" yaml:"buoy_binary_url"`
	// SSHUser is the default SSH user for reaching new nodes.
	SSHUser string `koanf:"ssh_user" yaml:"ssh_user"`
	// SSHPort is the default SSH port for reaching new nodes.
	SSHPort int `koanf:"ssh_port" yaml:"ssh_port"`
}

// LogConfig controls diagnostic logging.
type LogConfig struct {
	// Level is one of debug, info, warn, error.
	Level string `koanf:"level" yaml:"level"`
}

// UIConfig controls the embedded admin Web UI. It binds to localhost only —
// helm opens no inbound ports.
type UIConfig struct {
	// Listen is the localhost address the admin UI binds to.
	Listen string `koanf:"listen" yaml:"listen"`
}

// ProtocolsConfig toggles which data-plane protocols the fleet offers.
type ProtocolsConfig struct {
	AmneziaWG bool `koanf:"amneziawg" yaml:"amneziawg"`
	XRay      bool `koanf:"xray" yaml:"xray"`
}

// BeaconConfig controls the relay tier (DESIGN §2).
type BeaconConfig struct {
	// Embedded runs a beacon relay in-process inside helm.
	Embedded bool `koanf:"embedded" yaml:"embedded"`
	// Remote enables dialing out to remote beacon relays over a reverse tunnel.
	Remote bool `koanf:"remote" yaml:"remote"`
}

// AccountsConfig controls the account & profile-sync service (DESIGN §8).
type AccountsConfig struct {
	// Sync enables account login and E2E-encrypted profile sync.
	Sync bool `koanf:"sync" yaml:"sync"`
}

// RetentionConfig controls how long audit and metrics data is kept.
type RetentionConfig struct {
	AuditDays   int `koanf:"audit_days" yaml:"audit_days"`
	MetricsDays int `koanf:"metrics_days" yaml:"metrics_days"`
}

// RealityConfig controls the XRay REALITY decoy site (DESIGN §12).
type RealityConfig struct {
	DecoySite string `koanf:"decoy_site" yaml:"decoy_site"`
}

// FleetConfig holds fleet-shape defaults.
type FleetConfig struct {
	// Regions the operator intends to deploy into. Empty means "decide at
	// provisioning time" (personal: one, nearest).
	Regions []string `koanf:"regions" yaml:"regions"`
	// IdleNodes encourages pre-positioned, stopped nodes (enterprise).
	IdleNodes bool `koanf:"idle_nodes" yaml:"idle_nodes"`
	// VPNSubnet is the CIDR helm allocates per-device tunnel addresses from.
	VPNSubnet string `koanf:"vpn_subnet" yaml:"vpn_subnet"`
}
