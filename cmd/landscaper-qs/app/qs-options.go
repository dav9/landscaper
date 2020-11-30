package app

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v2"
)

type qsOptions struct {
	qsPath string

	workDir string

	kubeconfig string
	kubeconfigData string

	domain string
	
	repo struct {
		host string
		username string
		password string
		auth string
	}
}

type kubeconfigYAML struct {
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

	kYAML := kubeconfigYAML{}
	err = yaml.Unmarshal(data, &kYAML)
	if err != nil {
		return err
	}

	if len(kYAML.Clusters) == 0 && len(kYAML.Clusters[0].Cluster.Server) == 0 {
		return fmt.Errorf("invalid kubeconfig: no clusters found")
	}

	u, err := url.Parse(kYAML.Clusters[0].Cluster.Server)
	if err != nil {
		return err
	}
	
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