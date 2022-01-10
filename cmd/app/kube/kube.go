package kube

import (
	"context"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/fengxsong/toolkit/cmd/app/factory"
	"github.com/fengxsong/toolkit/cmd/app/options"
)

const name = "kube"

func init() {
	factory.Register(name, newSubCommand())
}

func newSubCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     name,
		Aliases: []string{"k8s"},
		Short:   "kubernetes toolkits",
	}
	cmd.AddCommand(newCheckCommand())
	cmd.AddCommand(newListCommand())
	options.AddKubeConfigFlags(cmd.PersistentFlags())
	return cmd
}

type client struct {
	kubeClient kubernetes.Interface
}

func newClient() (*client, error) {
	cfg, err := options.RESTClientGetter().ToRESTConfig()
	if err != nil {
		return nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &client{kubeClient}, nil
}

func (c *client) listDeploymentObjects(o *options.KubeListOption) ([]appsv1.Deployment, error) {
	list, err := c.kubeClient.AppsV1().Deployments(o.Namespace).List(context.Background(),
		metav1.ListOptions{FieldSelector: o.FieldSelector, LabelSelector: o.LabelSelector})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *client) listDeployments(o *options.KubeListOption) (deploymentList, error) {
	list, err := c.listDeploymentObjects(o)
	if err != nil {
		return nil, err
	}
	items := make(deploymentList, 0, len(list))
	for i := range list {
		items = append(items, fromBuiltinDeployment(list[i]))
	}
	return items, nil
}
