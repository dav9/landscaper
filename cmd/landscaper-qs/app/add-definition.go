package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

// NewAddDefinitionCommand creates command for adding definition
func NewAddDefinitionCommand(ctx context.Context, opts *qsOptions) *cobra.Command {
	addDefinition := &cobra.Command{
		Use:     "add-definition",
		Aliases: []string{"d"},
		Short:   "test",
		PreRunE: func(cmd *cobra.Command, args[]string) (err error) {
			if opts.kubeconfig == "" {
				return fmt.Errorf("kubeconfig is not set")
			}
			return opts.Load()
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if err = execute("helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx"); err != nil {
				return
			}

			if err = execute("helm repo update"); err != nil {
				return
			}

			if err = os.RemoveAll("/tmp/ingress-nginx"); err != nil {
				return
			}

			if err = execute("helm pull ingress-nginx/ingress-nginx --untar --destination /tmp"); err != nil {
				return
			}

			if err = execute(fmt.Sprintf("helm chart save /tmp/ingress-nginx %s/library/charts/ingress-nginx:0.0.1", opts.repo.host)); err != nil {
				return
			}
			
			if err = execute(fmt.Sprintf("helm registry login -u %s -p %s %s --kubeconfig %s", opts.repo.username, opts.repo.password, opts.repo.host, opts.kubeconfig)); err != nil {
				return
			}
			
			if err = execute(fmt.Sprintf("helm chart push %s/library/charts/ingress-nginx:0.0.1", opts.repo.host)); err != nil {
				return
			}
			
			if err = execute(fmt.Sprintf("docker login -u %s -p %s %s", opts.repo.username, opts.repo.password, opts.repo.host)); err != nil {
				return
			}
			
			if err = execute(fmt.Sprintf("landscaper-cli blueprints push %s/library/charts/ingress-nginx-blueprint:0.0.1 %s/definition/blueprint", opts.repo.host, opts.qsPath)); err != nil {
				return
			}
			
			templatePath := filepath.Join(opts.qsPath, "definition", "component-descriptor-template.yaml")
			resultPath := filepath.Join(opts.qsPath, "definition", "component-descriptor.yaml")
			
			t, err := template.ParseFiles(templatePath)
			f, err := os.Create(resultPath)
			if err != nil {
				return fmt.Errorf("cannot create component descriptor file: %v", err)
			}
			defer f.Close()

			err = t.Execute(f, map[string]string{"BaseURL": opts.repo.host + "/library/charts"})
			if err != nil {
				return fmt.Errorf("could not execute component descriptor template: %v", err)
			}
			
			if err = execute(fmt.Sprintf("landscaper-cli cd push %s/definition/component-descriptor.yaml", opts.qsPath)); err != nil {
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

	addDefinition.Flags().StringVar(&opts.kubeconfig, "kubeconfig", os.Getenv("KUBECONFIG"), "path to kubeconfig")
	addDefinition.Flags().StringVar(&opts.qsPath, "quick-start", filepath.Join(wd, "docs", "tutorials", "quick-start"), "path to quick-start directory")

	return addDefinition
}