package framework

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMapData makes it simpler to provide multiple data elements
// for a new ConfigMap
type ConfigMapData struct {
	Name string
	Data string
}

// CreateConfigMapData helper method to generate a ConfiMap using configuration Data
func (c *ContextData) CreateConfigMapData(name string, data ...ConfigMapData) (*v1.ConfigMap, error) {
	// In case no data element provided
	if data == nil || len(data) == 0 {
		return nil, fmt.Errorf("need at least one ConfigDataMap element")
	}
	dataMap := map[string]string{}
	for _, d := range data {
		dataMap[d.Name] = d.Data
	}

	cfgMap, err := c.Clients.KubeClient.CoreV1().ConfigMaps(c.Namespace).Create(context.TODO(), &v1.ConfigMap{
		ObjectMeta: v12.ObjectMeta{
			Name: name,
		},
		Data: dataMap,
	}, metav1.CreateOptions{})

	return cfgMap, err
}
