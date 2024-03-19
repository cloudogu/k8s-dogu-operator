package ecoSystem

import (
	"github.com/cloudogu/k8s-dogu-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type EcoSystemV1Alpha1Interface interface {
	Dogus(namespace string) DoguInterface
	DoguRestarts(namespace string) DoguRestartInterface
}

type EcoSystemV1Alpha1Client struct {
	restClient rest.Interface
}

func NewForConfig(c *rest.Config) (*EcoSystemV1Alpha1Client, error) {
	config := *c
	gv := schema.GroupVersion{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version}
	config.ContentConfig.GroupVersion = &gv
	config.APIPath = "/apis"

	s := scheme.Scheme
	err := v1.AddToScheme(s)
	if err != nil {
		return nil, err
	}

	metav1.AddToGroupVersion(s, gv)
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &EcoSystemV1Alpha1Client{restClient: client}, nil
}

func (c *EcoSystemV1Alpha1Client) Dogus(namespace string) DoguInterface {
	return &doguClient{
		client: c.restClient,
		ns:     namespace,
	}
}

func (c *EcoSystemV1Alpha1Client) DoguRestarts(namespace string) DoguRestartInterface {
	return &doguRestartClient{
		client: c.restClient,
		ns:     namespace,
	}
}
