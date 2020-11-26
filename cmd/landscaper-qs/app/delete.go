package app

import (
	"context"

	"github.com/spf13/cobra"
)


func NewDeleteCommand(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:     "delete",
		Aliases: []string{"d"},
		Short:   "delete landscaper",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}