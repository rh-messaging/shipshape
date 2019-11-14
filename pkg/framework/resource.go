package framework

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ResourceType int

const (
	Issuers ResourceType = iota
	Certificates
)

var (
	resourceMap = map[ResourceType]schema.GroupVersionResource{
		Issuers: {
			Group:    "certmanager.k8s.io",
			Version:  "v1alpha1",
			Resource: "issuers",
		},
		Certificates: {
			Group:    "certmanager.k8s.io",
			Version:  "v1alpha1",
			Resource: "certificates",
		},
	}
)

// GetResource returns the given resource type, identified by its given name
func (c *ContextData) GetResource(resourceType ResourceType, name string) (*unstructured.Unstructured, error) {
	return c.Clients.DynClient.Resource(resourceMap[resourceType]).Namespace(c.Namespace).Get(name, v1.GetOptions{})
}

// ListResources returns a list of resources found in the related Framework's namespace,
// for the given resource type
func (c *ContextData) ListResources(resourceType ResourceType) (*unstructured.UnstructuredList, error) {
	return c.Clients.DynClient.Resource(resourceMap[resourceType]).Namespace(c.Namespace).List(v1.ListOptions{})
}
