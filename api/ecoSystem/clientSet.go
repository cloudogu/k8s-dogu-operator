package ecoSystem

import (
	"github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type EcoSystemV2Interface interface {
	Dogus(namespace string) DoguInterface
	DoguRestarts(namespace string) DoguRestartInterface
}

type EcoSystemV2Client struct {
	restClient rest.Interface
}

func NewForConfig(c *rest.Config) (*EcoSystemV2Client, error) {
	config := *c
	gv := schema.GroupVersion{Group: v2.GroupVersion.Group, Version: v2.GroupVersion.Version}
	config.ContentConfig.GroupVersion = &gv
	config.APIPath = "/apis"

	s := scheme.Scheme
	err := v2.AddToScheme(s)
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

	return &EcoSystemV2Client{restClient: client}, nil
}

func (c *EcoSystemV2Client) Dogus(namespace string) DoguInterface {
	return &doguClient{
		client: c.restClient,
		ns:     namespace,
	}
}

func (c *EcoSystemV2Client) DoguRestarts(namespace string) DoguRestartInterface {
	return &doguRestartClient{
		client: c.restClient,
		ns:     namespace,
	}
}
