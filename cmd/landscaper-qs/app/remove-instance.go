package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// NewRemoveInstanceCommand creates command for removing instance
func NewRemoveInstanceCommand(ctx context.Context, opts *qsOptions) *cobra.Command {
	return &cobra.Command{
		Use:     "remove-instance",
		Short:   "remove instance of the definition",
		Run: func(cmd *cobra.Command, args []string) {
			err := execute(fmt.Sprintf("kubectl --kubeconfig %s delete installation my-ingress", opts.kubeconfig))
			if err != nil {
				
			}

			err = execute(fmt.Sprintf("kubectl --kubeconfig %s delete target my-cluster", opts.kubeconfig))
			if err != nil {
				fmt.Println(err.Error())
			}
		},
	}
}
