// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

package cli

import (
	"fmt"

	"github.com/PharosVPN/helm/internal/config"
	"github.com/PharosVPN/helm/internal/ssh"
	"github.com/spf13/cobra"
)

func newSSHKeyCmd() *cobra.Command {
	var cfgPath string
	cmd := &cobra.Command{
		Use:   "ssh-key",
		Short: "Print helm's SSH public key",
		Long: "Print helm's outbound SSH public key. Add this key to a new\n" +
			"node's ~/.ssh/authorized_keys (or the cloud provider's SSH keys)\n" +
			"before running `helm nodes add` — helm dials out with it to\n" +
			"install the buoy agent.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, conn, err := openState(cfgPath)
			if err != nil {
				return err
			}
			defer conn.Close()

			id, _, err := ssh.EnsureIdentity(cmd.Context(), conn)
			if err != nil {
				return err
			}
			fmt.Println(id.AuthorizedKey)
			return nil
		},
	}
	cmd.Flags().StringVar(&cfgPath, "config", config.DefaultPath, "path to the config file")
	return cmd
}
