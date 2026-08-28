package kubernetes

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

func (c *Client) sandboxResource() (dynamic.ResourceInterface, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	return c.dynamic.Resource(c.SandboxGVR()).Namespace(c.namespace), nil
}

func (c *Client) CreateSandbox(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	res, err := c.sandboxResource()
	if err != nil {
		return nil, err
	}
	out, err := res.Create(ctx, obj, metav1.CreateOptions{})
	return out, Translate(err)
}

func (c *Client) GetSandbox(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	res, err := c.sandboxResource()
	if err != nil {
		return nil, err
	}
	out, err := res.Get(ctx, name, metav1.GetOptions{})
	return out, Translate(err)
}

func (c *Client) DeleteSandbox(ctx context.Context, name string) error {
	res, err := c.sandboxResource()
	if err != nil {
		return err
	}
	return Translate(res.Delete(ctx, name, metav1.DeleteOptions{}))
}

func (c *Client) PatchSandbox(ctx context.Context, name string, patch []byte) (*unstructured.Unstructured, error) {
	res, err := c.sandboxResource()
	if err != nil {
		return nil, err
	}
	out, err := res.Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{})
	return out, Translate(err)
}
