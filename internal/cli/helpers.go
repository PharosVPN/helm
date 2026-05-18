// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

package cli

import (
	"context"
	"database/sql"
	"path/filepath"

	"github.com/PharosVPN/helm/internal/config"
	"github.com/PharosVPN/helm/internal/db"
	"github.com/PharosVPN/helm/internal/fleet"
	"github.com/PharosVPN/helm/internal/ssh"
)

// openState loads the config file and opens the migrated state database. The
// caller must Close the returned *sql.DB.
func openState(cfgPath string) (config.Config, *sql.DB, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return config.Config{}, nil, err
	}
	conn, err := db.Open(filepath.Join(cfg.StateDir, "app.db"))
	if err != nil {
		return config.Config{}, nil, err
	}
	if err := db.Migrate(conn); err != nil {
		conn.Close()
		return config.Config{}, nil, err
	}
	return cfg, conn, nil
}

// dialNew opens an SSH connection to a not-yet-enrolled node. The host key is
// trusted on first use and captured for later pinning.
func dialNew(ctx context.Context, conn *sql.DB, host, user string, port int) (*ssh.Conn, error) {
	id, _, err := ssh.EnsureIdentity(ctx, conn)
	if err != nil {
		return nil, err
	}
	return ssh.Dial(ctx, ssh.DialConfig{
		Host: host, Port: port, User: user, Signer: id.Signer,
	})
}

// dialNode opens an SSH connection to an enrolled node, verifying its pinned
// host key.
func dialNode(ctx context.Context, conn *sql.DB, node fleet.Node) (*ssh.Conn, error) {
	id, _, err := ssh.EnsureIdentity(ctx, conn)
	if err != nil {
		return nil, err
	}
	return ssh.Dial(ctx, ssh.DialConfig{
		Host:         node.SSHHost,
		Port:         node.SSHPort,
		User:         node.SSHUser,
		Signer:       id.Signer,
		KnownHostKey: node.SSHHostKey,
	})
}
