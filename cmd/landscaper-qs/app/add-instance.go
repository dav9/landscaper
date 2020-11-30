package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/spf13/cobra"
)

// NewAddInstanceCommand creates command for adding instance
func NewAddInstanceCommand(ctx context.Context, opts *qsOptions) *cobra.Command {
	addInstance := &cobra.Command{
		Use:     "add-instance",
		Aliases: []string{"i"},
		Short:   "adds instance of the definition",
		PreRunE: func(cmd *cobra.Command, args[]string) (err error) {
			if opts.kubeconfig == "" {
				return fmt.Errorf("kubeconfig is not set")
			}
			return opts.Load()
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			targetTemplatePath := filepath.Join(opts.qsPath, "instance", "target-template.yaml")
			targetResultPath := filepath.Join(opts.qsPath, "instance", "target.yaml")
			
			targetTemplate, err := template.New("target-template.yaml").Funcs(sprig.TxtFuncMap()).ParseFiles(targetTemplatePath)
			if err != nil {
				return err
			}
			targetFile, err := os.Create(targetResultPath)
			if err != nil {
				return fmt.Errorf("cannot create target file: %v", err)
			}
			defer targetFile.Close()

			err = targetTemplate.Execute(targetFile, map[string]string{"BaseURL": opts.repo.host + "/library/charts", "Kubeconfig": opts.kubeconfigData})
			if err != nil {
				return fmt.Errorf("cannot execute template: %v", err)
			}

			if err = execute(fmt.Sprintf("kubectl apply --kubeconfig %s -f %s", opts.kubeconfig, targetResultPath)); err != nil {
				return err
			}
			
			installationTemplatePath := filepath.Join(opts.qsPath, "instance", "installation-template.yaml")
			installationResultPath := filepath.Join(opts.qsPath, "instance", "installation.yaml")

			installationTemplate, err := template.ParseFiles(installationTemplatePath)
			installationFile, err := os.Create(installationResultPath)
			if err != nil {
				return fmt.Errorf("cannot create installation file: %v", err)
			}
			defer installationFile.Close()

			err = installationTemplate.Execute(installationFile, map[string]string{"BaseURL": opts.repo.host + "/library/charts"})
			if err != nil {
				return fmt.Errorf("could not execute installation template: %v", err)
			}

			if err = execute(fmt.Sprintf("kubectl apply --kubeconfig %s -f %s", opts.kubeconfig, installationResultPath)); err != nil {
				return err
			}
			
			return
		},
	}
	
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	addInstance.Flags().StringVar(&opts.kubeconfig, "kubeconfig", os.Getenv("KUBECONFIG"), "path to kubeconfig")
	addInstance.Flags().StringVar(&opts.qsPath, "quick-start", filepath.Join(wd, "docs", "tutorials", "quick-start"), "path to quick-start directory")

	return addInstance
}
