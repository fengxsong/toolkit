package kube

import (
	"io"
	"io/ioutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

// deployments list of deployment
type deploymentList []deployment

func (l deploymentList) Write(w io.Writer) error {
	b, err := yaml.Marshal(&l)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func (l deploymentList) ToMap() map[string]deployment {
	m := make(map[string]deployment, len(l))
	for _, item := range l {
		m[item.Key()] = item
	}
	return m
}

func getKeys(m map[string]deployment) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func loadFromFile(filename string) (deploymentList, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var list deploymentList
	if err = yaml.Unmarshal(b, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// deployment internal structure
type deployment struct {
	Name      string                                 `json:"name"`
	Namespace string                                 `json:"namespace"`
	Replicas  int                                    `json:"replicas"`
	Resources map[string]corev1.ResourceRequirements `json:"resources"`
	Skip      bool                                   `json:"skip,omitempty"`
}

func (d deployment) Key() string {
	return types.NamespacedName{Namespace: d.Namespace, Name: d.Name}.String()
}

// FromBuiltinDeployment ...
func fromBuiltinDeployment(obj appsv1.Deployment) deployment {
	dp := deployment{
		Name:      obj.ObjectMeta.Name,
		Namespace: obj.ObjectMeta.Namespace,
		Replicas:  int(*obj.Spec.Replicas),
		Resources: make(map[string]corev1.ResourceRequirements),
	}
	visitAll := func(containers []corev1.Container) {
		for i := range containers {
			dp.Resources[containers[i].Name] = containers[i].Resources
		}
	}
	for _, containers := range [][]corev1.Container{
		obj.Spec.Template.Spec.Containers,
		obj.Spec.Template.Spec.InitContainers,
	} {
		visitAll(containers)
	}
	return dp
}
