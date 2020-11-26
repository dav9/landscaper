package app

import (
	// "bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

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

			if err = createNamespace(ctx, opts, "ls-system"); err != nil {
				return err
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

	templatePath := filepath.Join(opts.workDir, "landscaper-values-template.yaml")
	resultPath := filepath.Join(opts.workDir, "landscaper-values.yaml")

	template, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	file, err := os.Create(resultPath)
	if err != nil {
		return fmt.Errorf("cannot create landscaper-values.yaml: %v", err)
	}
	defer file.Close()

	fmt.Println(map[string]string{"HarborCoreURL": opts.repo.host, "HarborAuth": opts.repo.auth})
	err = template.Execute(file, map[string]string{"HarborCoreURL": opts.repo.host, "HarborAuth": opts.repo.auth})
	if err != nil {
		return fmt.Errorf("could not execute installation template: %v", err)
	}

	return execute(fmt.Sprintf("helm upgrade --install --namespace ls-system landscaper %s --values %s --kubeconfig %s", 
		opts.chart, "landscaper-values.yaml", opts.kubeconfig))
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

	templatePath := filepath.Join(opts.workDir, "harbor-values-template.yaml")
	resultPath := filepath.Join(opts.workDir, "harbor-values.yaml")

	template, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	file, err := os.Create(resultPath)
	if err != nil {
		return fmt.Errorf("cannot create harbor-values.yaml: %v", err)
	}
	defer file.Close()

	err = template.Execute(file, map[string]string{"HarborCoreURL": opts.repo.host})
	if err != nil {
		return fmt.Errorf("could not execute installation template: %v", err)
	}

	return execute(fmt.Sprintf("helm upgrade --install --namespace ls-system harbor harbor/harbor --values %s --kubeconfig %s", "harbor-values.yaml", opts.kubeconfig))
}


func createNamespace(ctx context.Context, opts *qsOptions, name string) error {
	return execute(fmt.Sprintf("kubectl apply --filename %s --kubeconfig %s",
		filepath.Join(opts.workDir, "namespace.yaml"),
		opts.kubeconfig))

}
