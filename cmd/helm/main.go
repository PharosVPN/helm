// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

// Command helm is the PharosVPN controller / management plane.
package main

import (
	"fmt"
	"os"

	"github.com/PharosVPN/helm/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "helm: "+err.Error())
		os.Exit(1)
	}
}
