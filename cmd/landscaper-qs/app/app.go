// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"

	"github.com/gardener/landscaper/pkg/logger"
	"github.com/gardener/landscaper/pkg/version"

	"helm.sh/helm/v3/pkg/action"

	"helm.sh/helm/v3/pkg/cli"

	"github.com/spf13/cobra"
)

type qsOptions struct {
	workDir string
	kubeconfig string
	template string

	kubeconfigData string

	settings *cli.EnvSettings
	actionConfig *action.Configuration
	domain string

	chart string
	
	repo struct {
		host string
		username string
		password string
	}
}

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

			err = opts.Load()
			if err != nil {
				return fmt.Errorf("unable to setup: %v", err.Error())
			}

			return nil
		},
	}
	
	// settings.AddFlags(cmd.Flags())

	cmd.PersistentFlags().StringVarP(&opts.kubeconfig, "kubeconfig", "", "", "")
	if err := cobra.MarkFlagRequired(cmd.PersistentFlags(), "kubeconfig"); err != nil {
		return nil, err
	}

	logger.InitFlags(cmd.PersistentFlags())

	cmd.AddCommand(NewVersionCommand())
	cmd.AddCommand(NewInstallCommand(ctx, opts))
	cmd.AddCommand(NewTestCommand(ctx, opts))

	return cmd, nil
}

func (o *qsOptions) Load() (err error) {
	o.workDir, err = os.Getwd()
	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(o.kubeconfig)
	if err != nil {
		return err
	}

	o.kubeconfigData = string(data)

	os.Setenv("HELM_EXPERIMENTAL_OCI", "1")
	os.Setenv("HELM_NAMESPACE", "ls-system")
	o.settings = cli.New()
	o.settings.KubeConfig = o.kubeconfig

    o.actionConfig = new(action.Configuration)
	if err := o.actionConfig.Init(o.settings.RESTClientGetter(), 
		"ls-system", os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
        return err
	}

	//config.Host
	u, err := url.Parse("https://api.a.cp-k8s.shoot.canary.k8s-hana.ondemand.com")
	
	o.domain = u.Host[4:]

	o.repo.host = "h.ingress." + o.domain
	if len(o.repo.host) > 62 {
		err = fmt.Errorf("cannot install harbor: domain too long: len(%q) == %v", o.repo.host, len(o.repo.host))
		return err
	}

	o.repo.username = "admin"
	o.repo.password = "Harbor12345"

	return err
}

// NewVersionCommand returns the version of landscaper-qs
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
