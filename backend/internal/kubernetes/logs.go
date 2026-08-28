package kubernetes

import (
	"context"
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) OpenLogs(ctx context.Context, namespace, pod, container string, since *metav1.Time) (io.ReadCloser, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	opts := &corev1.PodLogOptions{
		Container:  container,
		Follow:     true,
		Timestamps: true,
	}
	if since != nil {
		opts.SinceTime = since
	} else {
		tail := int64(200)
		opts.TailLines = &tail
	}
	rc, err := c.kube.CoreV1().Pods(namespace).GetLogs(pod, opts).Stream(ctx)
	return rc, Translate(err)
}
