// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

// Package cli wires up the helm command-line interface.
package cli

import "github.com/spf13/cobra"

// version is the helm build version. Overridable at link time.
var version = "0.1.0-dev"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "helm",
		Short:         "PharosVPN controller",
		Long:          "helm — the PharosVPN controller and management plane.\n\nhelm is the source of truth for the fleet: it holds the CA, drives every\nVPN node over outbound mTLS, and serves the admin UI. It opens no inbound\nports.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newInitCmd(),
		newSSHKeyCmd(),
		newNodesCmd(),
	)
	return root
}

// Execute runs the helm CLI.
func Execute() error {
	return newRootCmd().Execute()
}
