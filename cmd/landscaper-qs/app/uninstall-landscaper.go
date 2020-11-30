package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// NewUninstallLandscaperCommand creates command for uninstalling landscaper
func NewUninstallLandscaperCommand(ctx context.Context, opts *qsOptions) *cobra.Command {
	return &cobra.Command{
		Use:     "uninstall-landscaper",
		Short:   "uninstalls landscaper",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if err = execute(fmt.Sprintf("helm uninstall harbor --namespace ls-system --kubeconfig %s", opts.kubeconfig)); err != nil {
				return
			}

			if err = execute(fmt.Sprintf("helm uninstall landscaper --namespace ls-system --kubeconfig %s", opts.kubeconfig)); err != nil {
				return
			}

			return
		},
	}
}