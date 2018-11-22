package factory

import (
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericclioptions/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/pkg/kubectl/cmd/util/openapi"
	openapivalidation "k8s.io/kubernetes/pkg/kubectl/cmd/util/openapi/validation"
	"k8s.io/kubernetes/pkg/kubectl/validation"
)

// Factory provides abstractions that allow the kubectl command to be extended across multiple types of resources and different API sets.
// The rings are here for a reason.
// In order for composers to be able to provide alternative factory implementations
// they need to provide low level pieces of *certain* functions so that when the factory calls back into itself
// it uses the custom version of the function. Rather than try to enumerate everything that someone would want to override
// we split the factory into rings, where each ring can depend on methods in an earlier ring, but cannot depend
// upon peer methods in its own ring.
// TODO: make the functions interfaces
type Factory interface {
	genericclioptions.RESTClientGetter

	// DynamicClient returns a dynamic client ready for use
	DynamicClient() (dynamic.Interface, error)

	// KubernetesClientSet gives you back an external clientset
	KubernetesClientSet() (*kubernetes.Clientset, error)

	// Returns a RESTClient for accessing Kubernetes resources or an error.
	RESTClient() (*restclient.RESTClient, error)

	// NewBuilder returns an object that assists in loading objects from both disk and the server
	// and which implements the common patterns for CLI interactions with generic resources.
	NewBuilder() *resource.Builder

	// Returns a RESTClient for working with the specified RESTMapping or an error. This is intended
	// for working with arbitrary resources and is not guaranteed to point to a Kubernetes APIServer.
	ClientForMapping(mapping *meta.RESTMapping) (resource.RESTClient, error)

	// Returns a RESTClient for working with Unstructured objects.
	UnstructuredClientForMapping(mapping *meta.RESTMapping) (resource.RESTClient, error)

	// Returns a schema that can validate objects stored on disk.
	Validator(validate bool) (validation.Schema, error)

	// OpenAPISchema returns the schema openapi schema definition
	OpenAPISchema() (openapi.Resources, error)
}

type factoryImpl struct {
	clientGetter genericclioptions.RESTClientGetter

	// openAPIGetter loads and caches openapi specs
	openAPIGetter openAPIGetter
}

type openAPIGetter struct {
	once   sync.Once
	getter openapi.Getter
}

func NewFactory(clientGetter genericclioptions.RESTClientGetter) Factory {
	if clientGetter == nil {
		panic("attempt to instantiate client_access_factory with nil clientGetter")
	}

	f := &factoryImpl{
		clientGetter: clientGetter,
	}

	return f
}

func (f *factoryImpl) ToRESTConfig() (*restclient.Config, error) {
	return f.clientGetter.ToRESTConfig()
}

func (f *factoryImpl) ToRESTMapper() (meta.RESTMapper, error) {
	return f.clientGetter.ToRESTMapper()
}

func (f *factoryImpl) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return f.clientGetter.ToDiscoveryClient()
}

func (f *factoryImpl) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return f.clientGetter.ToRawKubeConfigLoader()
}

func (f *factoryImpl) KubernetesClientSet() (*kubernetes.Clientset, error) {
	clientConfig, err := f.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(clientConfig)
}

func (f *factoryImpl) DynamicClient() (dynamic.Interface, error) {
	clientConfig, err := f.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(clientConfig)
}

// NewBuilder returns a new resource builder for structured api objects.
func (f *factoryImpl) NewBuilder() *resource.Builder {
	return resource.NewBuilder(f.clientGetter)
}

func (f *factoryImpl) RESTClient() (*restclient.RESTClient, error) {
	clientConfig, err := f.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	setKubernetesDefaults(clientConfig)
	return restclient.RESTClientFor(clientConfig)
}

func (f *factoryImpl) ClientForMapping(mapping *meta.RESTMapping) (resource.RESTClient, error) {
	cfg, err := f.clientGetter.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	if err := setKubernetesDefaults(cfg); err != nil {
		return nil, err
	}
	gvk := mapping.GroupVersionKind
	switch gvk.Group {
	case corev1.GroupName:
		cfg.APIPath = "/api"
	default:
		cfg.APIPath = "/apis"
	}
	gv := gvk.GroupVersion()
	cfg.GroupVersion = &gv
	return restclient.RESTClientFor(cfg)
}

func (f *factoryImpl) UnstructuredClientForMapping(mapping *meta.RESTMapping) (resource.RESTClient, error) {
	cfg, err := f.clientGetter.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	if err := restclient.SetKubernetesDefaults(cfg); err != nil {
		return nil, err
	}
	cfg.APIPath = "/apis"
	if mapping.GroupVersionKind.Group == corev1.GroupName {
		cfg.APIPath = "/api"
	}
	gv := mapping.GroupVersionKind.GroupVersion()
	cfg.ContentConfig = resource.UnstructuredPlusDefaultContentConfig()
	cfg.GroupVersion = &gv
	return restclient.RESTClientFor(cfg)
}

func (f *factoryImpl) Validator(validate bool) (validation.Schema, error) {
	if !validate {
		return validation.NullSchema{}, nil
	}

	resources, err := f.OpenAPISchema()
	if err != nil {
		return nil, err
	}

	return validation.ConjunctiveSchema{
		openapivalidation.NewSchemaValidation(resources),
		validation.NoDoubleKeySchema{},
	}, nil
}

// OpenAPISchema returns metadata and structural information about Kubernetes object definitions.
func (f *factoryImpl) OpenAPISchema() (openapi.Resources, error) {
	discovery, err := f.clientGetter.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	// Lazily initialize the OpenAPIGetter once
	f.openAPIGetter.once.Do(func() {
		// Create the caching OpenAPIGetter
		f.openAPIGetter.getter = openapi.NewOpenAPIGetter(discovery)
	})

	// Delegate to the OpenAPIGetter
	return f.openAPIGetter.getter.Get()
}
