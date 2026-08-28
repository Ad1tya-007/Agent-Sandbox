package kubernetes

import "testing"

func TestNewMissingKubeconfig(t *testing.T) {
	c := New("/definitely/not/a/kubeconfig", "custom-ns")
	if c == nil {
		t.Fatal("New returned nil")
	}
	if c.Err() == nil {
		t.Fatal("expected error loading missing kubeconfig")
	}
	if c.Namespace() != "custom-ns" {
		t.Fatalf("Namespace() = %q, want custom-ns", c.Namespace())
	}
	if c.ClusterName() != "" {
		t.Fatalf("ClusterName() = %q, want empty", c.ClusterName())
	}
	if c.RESTConfig() != nil {
		t.Fatal("RESTConfig() = non-nil, want nil")
	}
}

func TestNewMissingKubeconfigDefaultNamespace(t *testing.T) {
	c := New("/definitely/not/a/kubeconfig", "")
	if c.Err() == nil {
		t.Fatal("expected error loading missing kubeconfig")
	}
	if c.Namespace() != "default" {
		t.Fatalf("Namespace() = %q, want default", c.Namespace())
	}
}
