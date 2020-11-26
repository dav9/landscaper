package app

import (
	// "bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"github.com/gofrs/flock"
	"log"
	"time"
	"strings"
	"encoding/base64"

	"gopkg.in/yaml.v2"
	"io/ioutil"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	vals := landscaperVals(opts)
	
	chart, err := loader.Load(opts.chart)
	if err != nil {
		return err
	}

	var r *release.Release
	status := action.NewStatus(opts.actionConfig)
	_, err = status.Run("landscaper")
	if err != nil {
		if err.Error() == "release: not found" {
			install := action.NewInstall(opts.actionConfig)
			install.Namespace = "ls-system"
			install.ReleaseName = "landscaper"

			r, err = install.Run(chart, vals)
			if err != nil {
				return err
			}
			fmt.Println("release install: ", r.Info.Status)
		} else {
			return err
		}
	} else {
		upgrade := action.NewUpgrade(opts.actionConfig)
		upgrade.Namespace = "ls-system"
	
		r, err = upgrade.Run("landscaper", chart, vals)
		if err != nil {
			return err
		}
	
		fmt.Println("release upgrade: ", r.Info.Status)
	}

	if f != nil {
		_, err = f.WriteString(r.Manifest)
		if err != nil {
			return
		}
	}

	return nil
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

	repoAdd(opts, "harbor", "https://helm.goharbor.io")
	repoUpdate(opts, "harbor")

	client := action.NewInstall(opts.actionConfig)

	cp, err := client.ChartPathOptions.LocateChart(fmt.Sprintf("%s/%s", "harbor", "harbor"), opts.settings)
	if err != nil {
		log.Fatal(err)
	}

	chart, err := loader.Load(cp)
	if err != nil {
		return
	}

	vals := harborVals(opts)

	var r *release.Release
	status := action.NewStatus(opts.actionConfig)
	_, err = status.Run("harbor")
	if err != nil {
		if err.Error() == "release: not found" {
			install := action.NewInstall(opts.actionConfig)
			install.Namespace = "ls-system"
			install.ReleaseName = "harbor"
			r, err = install.Run(chart, vals)
			if err != nil {
				return 
			}
			fmt.Println("release install: ", r.Info.Status)
		} else {
			return 
		}
	} else {
		upgrade := action.NewUpgrade(opts.actionConfig)
		upgrade.Namespace = "ls-system"
	
		r, err = upgrade.Run("harbor", chart, vals)
		if err != nil {
			return 
		}
	
		fmt.Println("release upgrade: ", r.Info.Status)
	}

	if f != nil {
		_, err = f.WriteString(r.Manifest)
		if err != nil {
			return
		}
	}

	return nil
}

func harborVals(opts *qsOptions) map[string]interface{} {
	return map[string]interface{} {
		"externalURL": "https://" + opts.repo.host,
		"harborAdminPassword": opts.repo.password,
		"chartmuseum": map[string]interface{} {"enabled": false},
		"clair": map[string]interface{} {"enabled": false},
		"trivy": map[string]interface{} {"enabled": false},
		"notary": map[string]interface{} {"enabled": false},
		"expose": map[string]interface{} {
			"tls": map[string]interface{} {
				"certSource": "secret",
				"secret": map[string]interface{} {
					"secretName": "harbor-tls-secret",
				},
			},
			"ingress": map[string]interface{} {
				"annotations": map[string]interface{} {
					"dns.gardener.cloud/class": "garden",
					"dns.gardener.cloud/dnsnames": opts.repo.host,
					"cert.gardener.cloud/purpose": "managed",
				},
				"hosts": map[string]interface{} {
					"core": opts.repo.host,
				},
			},
		},
	}
}

func createNamespace(ctx context.Context, opts *qsOptions, name string) error {
	clientset := opts.clientset

	if _, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{}); 
		errors.IsNotFound(err) {
		nsSpec := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}

		if _, err := clientset.CoreV1().Namespaces().Create(ctx, nsSpec, metav1.CreateOptions{}); 
			!errors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

func repoAdd(opts *qsOptions, name, url string) {
	repoFile := opts.settings.RepositoryConfig

	//Ensure the file directory exists as it is required for file locking
	err := os.MkdirAll(filepath.Dir(repoFile), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}

	// Acquire a file lock for process synchronization
	fileLock := flock.New(strings.Replace(repoFile, filepath.Ext(repoFile), ".lock", 1))
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	locked, err := fileLock.TryLockContext(lockCtx, time.Second)
	if err == nil && locked {
		defer fileLock.Unlock()
	}
	if err != nil {
		log.Fatal(err)
	}

	b, err := ioutil.ReadFile(repoFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		log.Fatal(err)
	}

	if f.Has(name) {
		fmt.Printf("repository name (%s) already exists\n", name)
		return
	}

	c := repo.Entry{
		Name: name,
		URL:  url,
	}

	r, err := repo.NewChartRepository(&c, getter.All(opts.settings))
	if err != nil {
		log.Fatal(err)
	}

	if _, err := r.DownloadIndexFile(); err != nil {
		err := fmt.Errorf("looks like %q is not a valid chart repository or cannot be reached: %v", url, err)
		log.Fatal(err)
	}

	f.Update(&c)

	if err := f.WriteFile(repoFile, 0644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%q has been added to your repositories\n", name)
}

func repoUpdate(opts *qsOptions, name string) error {
	repoFile := opts.settings.RepositoryConfig

	f, err := repo.LoadFile(repoFile)
	if err != nil || len(f.Repositories) == 0 {
		return fmt.Errorf("no repositories found: %v", err)
	}

	repoFound := false
	for _, cfg := range f.Repositories {
		if cfg.Name == "name" {
			repoFound = true

			r, err := repo.NewChartRepository(cfg, getter.All(opts.settings))
			if err != nil {
				return err
			}

			_, err = r.DownloadIndexFile()
			if err != nil {
				return err
			}
			fmt.Printf("%s repository updated\n", name)
		}
	}
	
	if !repoFound {
		return fmt.Errorf("could not update %s repository: repository not added", name)
	}

	return nil
}