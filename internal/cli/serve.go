// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

package cli

import (
	"fmt"
	"sync"

	"github.com/PharosVPN/helm/internal/config"
	"github.com/PharosVPN/helm/internal/fleet"
	"github.com/PharosVPN/helm/internal/live"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var cfgPath string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the live plane and localhost admin server",
		Long: "Run helm's live plane (DESIGN §7): hold a WatchEvents stream\n" +
			"open to every enrolled node and fan events out to admin browsers\n" +
			"over a localhost WebSocket. Runs until interrupted.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			cfg, conn, err := openState(cfgPath)
			if err != nil {
				return err
			}
			defer conn.Close()

			dialer, err := newControlDialer(ctx, conn)
			if err != nil {
				return err
			}
			nodes, err := fleet.ListNodes(ctx, conn)
			if err != nil {
				return err
			}

			hub := live.NewHub()
			var wg sync.WaitGroup
			watched := 0
			for _, n := range nodes {
				if n.ControlAddr == "" {
					continue
				}
				watched++
				wg.Add(1)
				go func(node fleet.Node) {
					defer wg.Done()
					live.WatchNode(ctx, dialer, node, hub)
				}(n)
			}

			srv := live.NewServer(cfg.UI.Listen, hub)
			fmt.Printf("helm live plane — http://%s, watching %d node(s)\n", cfg.UI.Listen, watched)
			fmt.Printf("  events:  ws://%s/ws/events\n", cfg.UI.Listen)

			err = srv.Run(ctx)
			wg.Wait()
			return err
		},
	}
	cmd.Flags().StringVar(&cfgPath, "config", config.DefaultPath, "path to the config file")
	return cmd
}
