package framework

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/ghodss/yaml"
	"github.com/rh-messaging/shipshape/pkg/framework/log"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ResourceType int

const (
	Issuers ResourceType = iota
	Certificates
	Deployments
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
		Deployments: {Group: "apps", Version: "v1", Resource: "deployments"},
	}
)

// GetResource returns the given resource type, identified by its given name
func (c *ContextData) GetResource(resourceType ResourceType, name string) (*unstructured.Unstructured, error) {
	return c.Clients.DynClient.Resource(resourceMap[resourceType]).Namespace(c.Namespace).Get(context.TODO(), name, v1.GetOptions{})
}
func (c *ContextData) GetResourceGroupVersion(gv schema.GroupVersionResource, name string) (*unstructured.Unstructured, error) {
	return c.Clients.DynClient.Resource(gv).Namespace(c.Namespace).Get(context.TODO(), name, v1.GetOptions{})
}

// ListResources returns a list of resources found in the related Framework's namespace,
// for the given resource type
func (c *ContextData) ListResources(resourceType ResourceType) (*unstructured.UnstructuredList, error) {
	return c.Clients.DynClient.Resource(resourceMap[resourceType]).Namespace(c.Namespace).List(context.TODO(), v1.ListOptions{})
}
func (c *ContextData) ListResourcesGroupVersion(gv schema.GroupVersionResource) (*unstructured.UnstructuredList, error) {
	return c.Clients.DynClient.Resource(gv).Namespace(c.Namespace).List(context.TODO(), v1.ListOptions{})
}

// CreateResource creates a resource based on provided (known) resource type and unstructured data
func (c *ContextData) CreateResource(resourceType ResourceType, obj *unstructured.Unstructured, options v1.CreateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return c.Clients.DynClient.Resource(resourceMap[resourceType]).Namespace(c.Namespace).Create(context.TODO(), obj, options, subresources...)
}
func (c *ContextData) CreateResourceGroupVersion(gv schema.GroupVersionResource, obj *unstructured.Unstructured, options v1.CreateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return c.Clients.DynClient.Resource(gv).Namespace(c.Namespace).Create(context.TODO(), obj, options, subresources...)
}

// DeleteResource deletes a resource based on provided (known) resource type and name
func (c *ContextData) DeleteResource(resourceType ResourceType, name string, options v1.DeleteOptions, subresources ...string) error {
	return c.Clients.DynClient.Resource(resourceMap[resourceType]).Namespace(c.Namespace).Delete(context.TODO(), name, options, subresources...)
}
func (c *ContextData) DeleteResourceGroupVersion(gv schema.GroupVersionResource, name string, options v1.DeleteOptions, subresources ...string) error {
	return c.Clients.DynClient.Resource(gv).Namespace(c.Namespace).Delete(context.TODO(), name, options, subresources...)
}

func LoadYamlFromUrl(url string) (*unstructured.Unstructured, error) {
	var unsObj unstructured.Unstructured

	resp, err := http.Get(url) //load yaml body from url
	if err != nil {
		log.Logf("error during loading %s: %v", url, err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Logf("error during loading %s: %v", url, err)
		return nil, err
	}
	jsonBody, err := yaml.YAMLToJSON(body)
	if err != nil {
		log.Logf("error during parsing %s: %v", url, err)
		return nil, err
	}

	err = json.Unmarshal(jsonBody, &unsObj)
	return &unsObj, err
}
