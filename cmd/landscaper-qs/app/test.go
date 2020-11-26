package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/spf13/cobra"
)


func NewTestCommand(ctx context.Context, opts *qsOptions) *cobra.Command {
	return &cobra.Command{
		Use:     "test",
		Aliases: []string{"t"},
		Short:   "test",
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
			
			if err = execute(fmt.Sprintf("landscaper-cli blueprints push %s/library/charts/ingress-nginx-blueprint:0.0.1 definition/blueprint", opts.repo.host)); err != nil {
				return
			}
			
			path, err := os.Getwd()
			if err != nil {
				return
			}
			
			templatePath := filepath.Join(path, "definition", "component-descriptor-template.yaml")
			resultPath := filepath.Join(path, "definition", "component-descriptor.yaml")
			
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
			
			if err = execute("landscaper-cli cd push definition/component-descriptor.yaml"); err != nil {
				return err
			}
			
			targetTemplatePath := filepath.Join(path, "instance", "target-template.yaml")
			targetResultPath := filepath.Join(path, "instance", "target.yaml")
			
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
			
			installationTemplatePath := filepath.Join(path, "instance", "installation-template.yaml")
			installationResultPath := filepath.Join(path, "instance", "installation.yaml")

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
}

func execute(command string) (err error) {
	fmt.Printf("executing: %s\n", command)

	arr := strings.Split(command, " ")

	c := exec.Command(arr[0], arr[1:]...)
	c.Env = []string{"HELM_EXPERIMENTAL_OCI=1", "HOME=" + os.Getenv("HOME"), "PATH=" + os.Getenv("PATH")}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	err = c.Run()
	
	fmt.Println()

	return
}