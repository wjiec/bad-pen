package machinery

import (
	"flag"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func LoadConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()

		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	}
	return config, err
}

func NewClientset() (*kubernetes.Clientset, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}
