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

type qsInstallOptions struct {
	chart string

	template string

	*qsOptions
}

// NewInstallLandscaperCommand creates command for installing landscaper
func NewInstallLandscaperCommand(ctx context.Context, opts *qsOptions) *cobra.Command {
	qsInstallOpts := &qsInstallOptions{qsOptions: opts}
	install := &cobra.Command{
		Use:     "install-landscaper",
		Aliases: []string{"l"},
		Short:   "install landscaper",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if opts.kubeconfig == "" {
				return fmt.Errorf("kubeconfig is not set")
			}
			err = opts.Load()
			if err != nil {
				return fmt.Errorf("unable to setup: %v", err.Error())
			}

			var f *os.File
			if qsInstallOpts.template != "" {
				f, err = os.OpenFile(qsInstallOpts.template, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
				if err != nil {
					return
				}
				defer f.Close()
			}

			if err = createNamespace(ctx, qsInstallOpts, "ls-system"); err != nil {
				return err
			}

			err = addHarbor(ctx, qsInstallOpts, f)
			if err != nil {
				return err
			}

			err = addLandscaper(ctx, qsInstallOpts, f)
			if err != nil {
				return err
			}

			return nil
		},
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	install.Flags().StringVar(&opts.kubeconfig, "kubeconfig", os.Getenv("KUBECONFIG"), "path to kubeconfig")
	install.Flags().StringVar(&opts.qsPath, "quick-start", filepath.Join(wd, "docs", "tutorials", "quick-start"), "path to quick-start directory")

	install.Flags().StringVarP(&qsInstallOpts.chart, "chart", "", "./charts/landscaper", "path to chart landscaper")

	return install
}

func createNamespace(ctx context.Context, opts *qsInstallOptions, name string) error {
	return execute(fmt.Sprintf("kubectl apply --filename %s --kubeconfig %s",
		filepath.Join(opts.qsPath, "namespace.yaml"),
		opts.kubeconfig))
}

func addHarbor(ctx context.Context, opts *qsInstallOptions, f *os.File) (err error) {
	fmt.Println("installing harbor")

	if err = execute("helm repo add harbor https://helm.goharbor.io"); err != nil {
		return err
	}

	if err = execute("helm repo update"); err != nil {
		return err
	}

	templatePath := filepath.Join(opts.qsPath, "harbor-values-template.yaml")
	resultPath := filepath.Join(opts.qsPath, "harbor-values.yaml")

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

	return execute(fmt.Sprintf("helm upgrade --install --namespace ls-system harbor harbor/harbor --values %s --kubeconfig %s", resultPath, opts.kubeconfig))
}

func addLandscaper(ctx context.Context, opts *qsInstallOptions, f *os.File) (err error) {
	fmt.Println("installing landscaper")

	templatePath := filepath.Join(opts.qsPath, "landscaper-values-template.yaml")
	resultPath := filepath.Join(opts.qsPath, "landscaper-values.yaml")

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
		opts.chart, resultPath, opts.kubeconfig))
}