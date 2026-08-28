package kubernetes

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) ListPods(ctx context.Context, namespace, selector string) (*corev1.PodList, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	out, err := c.kube.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	return out, Translate(err)
}

func (c *Client) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	out, err := c.kube.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	return out, Translate(err)
}

func (c *Client) GetPVC(ctx context.Context, namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	out, err := c.kube.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	return out, Translate(err)
}

func (c *Client) GetService(ctx context.Context, namespace, name string) (*corev1.Service, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	out, err := c.kube.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	return out, Translate(err)
}

func (c *Client) ListPVCs(ctx context.Context, namespace, selector string) (*corev1.PersistentVolumeClaimList, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	out, err := c.kube.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	return out, Translate(err)
}

func (c *Client) ListServices(ctx context.Context, namespace, selector string) (*corev1.ServiceList, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	out, err := c.kube.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	return out, Translate(err)
}

func (c *Client) ListEvents(ctx context.Context, namespace, fieldSelector string) (*corev1.EventList, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	out, err := c.kube.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{FieldSelector: fieldSelector})
	return out, Translate(err)
}
