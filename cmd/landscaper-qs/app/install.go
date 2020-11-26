package app

import (
	// "bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func NewInstallCommand(ctx context.Context, opts *qsOptions) *cobra.Command {
	install := &cobra.Command{
		Use:     "install",
		Aliases: []string{"i"},
		Short:   "install landscaper",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var f *os.File
			if opts.template != "" {
				f, err = os.OpenFile(opts.template, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
				if err != nil {
					return
				}
				defer f.Close()
			}

			err = addHarborRepo(ctx, opts, f)
			if err != nil {
				return err
			}

			err = addLandscaper(ctx, opts, f)
			if err != nil {
				return err
			}

			return nil
		},
	}

	install.Flags().StringVarP(&opts.template, "template", "", "", "")
	install.Flags().StringVarP(&opts.chart, "chart", "", "./chart/landscaper", "path to chart landscaper")

	return install
}

func addLandscaper(ctx context.Context, opts *qsOptions, f *os.File) (err error) {
	fmt.Println("installing landscaper")
	if err = createNamespace(ctx, opts, "ls-system"); err != nil {
		return err
	}

	return execute(fmt.Sprintf("helm upgrade --install --namespace ls-system landscaper %s --values %s", opts.chart, "landscaper-values.yaml"))

}

func landscaperVals(opts *qsOptions) map[string]interface{} {
	auths := map[string]interface{} {
		opts.repo.host: map[string]interface{} {
			"auth": base64.StdEncoding.EncodeToString([]byte(opts.repo.username + ":" + opts.repo.password)),
		},
	}

	vals := map[string]interface{} {
		"landscaper": map[string]interface{} {
			"registryConfig": map[string]interface{} {
				"blueprints": map[string]interface{} {
					"secrets": map[string]interface{} {
						"default": map[string]interface{} {
							"auths": auths,
						},
					},
				},
				"components": map[string]interface{} {
					"secrets": map[string]interface{} {
						"default": map[string]interface{} {
							"auths": auths,
						},
					},
				},
			},
			"deployers": []string{"container", "helm", "mock"},
		},
		"image": map[string]string {
			"repository": "eu.gcr.io/gardener-project/landscaper/landscaper-controller",
			"pullPolicy": "IfNotPresent",
			"tag": "0.2.0-dev-89c8305937b649f93553bc59d3e935ce7a40913f",
		},
	}

	return vals
}

func addHarborRepo(ctx context.Context, opts *qsOptions, f *os.File) (err error) {
	fmt.Println("installing harbor")
	if err = createNamespace(ctx, opts, "ls-system"); err != nil {
		return 
	}

	if err = execute("helm repo add harbor https://helm.goharbor.io"); err != nil {
		return err
	}

	if err = execute("helm repo update"); err != nil {
		return err
	}

	return execute(fmt.Sprintf("helm upgrade --install --namespace ls-system harbor harbor/harbor --values %s", "harbor-values.yaml"))
}


func createNamespace(ctx context.Context, opts *qsOptions, name string) error {
	return execute(fmt.Sprintf("kubectl apply --filename %s --kubeconfig %s",
		filepath.Join(opts.workDir, "namespace.yaml"),
		opts.kubeconfig))

}
