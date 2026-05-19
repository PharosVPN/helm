// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

package cli

import (
	"fmt"
	"sync"

	"github.com/PharosVPN/helm/internal/api"
	"github.com/PharosVPN/helm/internal/auth"
	"github.com/PharosVPN/helm/internal/config"
	"github.com/PharosVPN/helm/internal/fleet"
	"github.com/PharosVPN/helm/internal/live"
	"github.com/PharosVPN/helm/internal/profile"
	"github.com/PharosVPN/helm/internal/provision"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var cfgPath string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the admin server and live plane",
		Long: "Run helm's admin server: the localhost JSON API and admin UI,\n" +
			"plus the live plane (DESIGN §7) — a WatchEvents stream held open\n" +
			"to every enrolled node, fanned out to admin browsers over a\n" +
			"WebSocket. Runs until interrupted.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			cfg, conn, err := openState(cfgPath)
			if err != nil {
				return err
			}
			defer conn.Close()

			// The config password is the source of truth for the fixed admin.
			if err := auth.SyncConfigAdmin(ctx, conn, cfg.Admin.Password); err != nil {
				return err
			}

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

			provOpts := provision.Options{
				VPNSubnet: cfg.Fleet.VPNSubnet,
				PortMin:   cfg.Fleet.EndpointPortMin,
				PortMax:   cfg.Fleet.EndpointPortMax,
				Rotation: profile.RotationPolicy{
					Enabled:         cfg.Fleet.Rotation.Enabled,
					IntervalSeconds: cfg.Fleet.Rotation.IntervalSeconds,
					JitterSeconds:   cfg.Fleet.Rotation.JitterSeconds,
				},
			}
			srv := api.NewServer(cfg.UI.Listen, conn, hub, provOpts)
			fmt.Printf("helm admin server — http://%s, watching %d node(s)\n", cfg.UI.Listen, watched)
			fmt.Printf("  api:     http://%s/api\n", cfg.UI.Listen)
			fmt.Printf("  events:  ws://%s/ws/events\n", cfg.UI.Listen)

			err = srv.Run(ctx)
			wg.Wait()
			return err
		},
	}
	cmd.Flags().StringVar(&cfgPath, "config", config.DefaultPath, "path to the config file")
	return cmd
}
