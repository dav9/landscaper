// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"

	"github.com/gardener/landscaper/pkg/logger"
	"github.com/gardener/landscaper/pkg/version"

	"github.com/spf13/cobra"
)

// NewLandscaperQSCommand returns the quick start root command for landscaper
func NewLandscaperQSCommand(ctx context.Context) (*cobra.Command, error) {
	opts := &qsOptions{}
	cmd := &cobra.Command{
		Use:   "landscaper-qs",
		Short: "landscaper quick start",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			log, err := logger.NewCliLogger()
			if err != nil {
				return fmt.Errorf("unable to setup logger: %v", err.Error())
			}
			logger.SetLogger(log)

			return nil
		},
	}

	// cmd.PersistentFlags().StringVar(&opts.kubeconfig, "kubeconfig", os.Getenv("KUBECONFIG"), "")
	// cmd.PersistentFlags().StringVar(&opts.qsPath, "quick-start", filepath.Join(wd, "docs", "tutorials", "quick-start"), "")

	logger.InitFlags(cmd.PersistentFlags())

	cmd.AddCommand(NewVersionCommand())
	cmd.AddCommand(NewInstallLandscaperCommand(ctx, opts))
	cmd.AddCommand(NewUninstallLandscaperCommand(ctx, opts))
	cmd.AddCommand(NewAddDefinitionCommand(ctx, opts))
	cmd.AddCommand(NewAddInstanceCommand(ctx, opts))
	cmd.AddCommand(NewRemoveInstanceCommand(ctx, opts))

	return cmd, nil
}

// NewVersionCommand creates command for the version of landscaper-qs
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "displays the version",
		Run: func(cmd *cobra.Command, args []string) {
			v := version.Get()
			fmt.Printf("%#v", v)
		},
	}
}
