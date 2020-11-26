// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"

	"github.com/gardener/landscaper/pkg/logger"
	"github.com/gardener/landscaper/pkg/version"
	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
)

type qsOptions struct {
	workDir string
	kubeconfig string
	template string

	kubeconfigData string

	domain string

	chart string
	
	repo struct {
		host string
		username string
		password string
		auth string
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

type KubeconfigYAML struct {
	Clusters []struct {
		Cluster struct {
			Server string
		}
	}
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

	kubeconfigYaml := KubeconfigYAML{}
	err = yaml.Unmarshal(data, &kubeconfigYaml)
	if err != nil {
		return err
	}

	if len(kubeconfigYaml.Clusters) == 0 && len(kubeconfigYaml.Clusters[0].Cluster.Server) == 0 {
		return fmt.Errorf("invalid kubeconfig: no clusters found")
	}

	u, err := url.Parse(kubeconfigYaml.Clusters[0].Cluster.Server)
	
	o.domain = u.Host[4:]

	o.repo.host = "h.ingress." + o.domain
	if len(o.repo.host) > 62 {
		err = fmt.Errorf("cannot install harbor: domain too long: len(%q) == %v", o.repo.host, len(o.repo.host))
		return err
	}

	o.repo.username = "admin"
	o.repo.password = "Harbor12345"

	o.repo.auth = base64.StdEncoding.EncodeToString([]byte(o.repo.username + ":" + o.repo.password))

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
