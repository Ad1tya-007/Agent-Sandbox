package kubernetes

import (
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	SandboxGroup    = "agents.x-k8s.io"
	SandboxVersion  = "v1beta1"
	SandboxResource = "sandboxes"
	SandboxKind     = "Sandbox"
)

func SandboxGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    SandboxGroup,
		Version:  SandboxVersion,
		Resource: SandboxResource,
	}
}

type Client struct {
	kube      kubernetes.Interface
	dynamic   dynamic.Interface
	rest      *rest.Config
	namespace string
	cluster   string
	gvr       schema.GroupVersionResource
	err       error
}

func New(kubeconfigPath, namespace string) *Client {
	c := &Client{namespace: "default", gvr: SandboxGVR()}
	if namespace != "" {
		c.namespace = namespace
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		c.err = fmt.Errorf("load kubeconfig: %w", err)
		return c
	}

	if raw, err := clientConfig.RawConfig(); err == nil {
		c.cluster = raw.CurrentContext
	}

	if namespace == "" {
		if ns, _, err := clientConfig.Namespace(); err == nil && ns != "" {
			c.namespace = ns
		}
	}

	kube, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		c.err = fmt.Errorf("kubernetes client: %w", err)
		return c
	}
	dyn, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		c.err = fmt.Errorf("dynamic client: %w", err)
		return c
	}

	c.kube = kube
	c.dynamic = dyn
	c.rest = restConfig
	c.gvr = SandboxGVR()
	return c
}

func (c *Client) Err() error {
	if c == nil {
		return nil
	}
	return c.err
}

func (c *Client) Namespace() string {
	if c == nil {
		return ""
	}
	return c.namespace
}

func (c *Client) ClusterName() string {
	if c == nil {
		return ""
	}
	return c.cluster
}

func (c *Client) RESTConfig() *rest.Config {
	if c == nil {
		return nil
	}
	return c.rest
}

func (c *Client) SandboxGVR() schema.GroupVersionResource {
	if c != nil && c.gvr.Resource != "" {
		return c.gvr
	}
	return SandboxGVR()
}

func (c *Client) Kube() kubernetes.Interface {
	if c == nil {
		return nil
	}
	return c.kube
}

func (c *Client) Dynamic() dynamic.Interface {
	if c == nil {
		return nil
	}
	return c.dynamic
}

func (c *Client) ready() error {
	if c == nil {
		return errors.New("kubernetes client is nil")
	}
	if c.err != nil {
		return c.err
	}
	if c.kube == nil || c.dynamic == nil {
		return errors.New("kubernetes client not initialized")
	}
	return nil
}
