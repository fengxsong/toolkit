package options

import (
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var configFlags = genericclioptions.NewConfigFlags(true)

// AddKubeConfigFlags add kube configflags to flagset
func AddKubeConfigFlags(fs *pflag.FlagSet) {
	configFlags.AddFlags(fs)
}

// RESTClientGetter
func RESTClientGetter() genericclioptions.RESTClientGetter {
	return configFlags
}

type KubeListOption struct {
	Namespace     string
	FieldSelector string
	LabelSelector string
}

func (o *KubeListOption) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.Namespace, "namespace", "n", metav1.NamespaceAll, "Namespace to list")
	fs.StringVar(&o.FieldSelector, "field-selector", "", "Fieldselector")
	fs.StringVar(&o.LabelSelector, "label-selector", "", "Lableselector")
}
