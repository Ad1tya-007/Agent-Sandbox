package sandbox

import (
	"context"
	"encoding/json"
	"errors"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/kubernetes"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
)

type Store interface {
	Namespace() string
	CreateSandbox(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
	GetSandbox(ctx context.Context, name string) (*unstructured.Unstructured, error)
	PatchSandbox(ctx context.Context, name string, patch []byte) (*unstructured.Unstructured, error)
	DeleteSandbox(ctx context.Context, name string) error
}

type Service struct {
	k8s Store
}

func New(k8s Store) *Service {
	return &Service{k8s: k8s}
}

func (s *Service) Create(ctx context.Context, in models.CreateInput) (*models.CreateResult, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if _, err := s.k8s.CreateSandbox(ctx, FromCreate(s.k8s.Namespace(), in)); err != nil {
		return nil, mapStoreError(err, in.Name)
	}
	return &models.CreateResult{Name: in.Name}, nil
}

func (s *Service) Pause(ctx context.Context, name string) error {
	obj, err := s.get(ctx, name)
	if err != nil {
		return err
	}
	if Phase(obj) != models.PhaseRunning {
		return models.ErrPauseNotRunning
	}
	return s.patchSpec(ctx, name,
		map[string]any{"operatingMode": "Suspended"},
		map[string]any{"replicas": int64(0)},
	)
}

func (s *Service) Resume(ctx context.Context, name string) error {
	obj, err := s.get(ctx, name)
	if err != nil {
		return err
	}
	if Phase(obj) != models.PhasePaused {
		return models.ErrResumeNotPaused
	}
	return s.patchSpec(ctx, name,
		map[string]any{"operatingMode": "Running"},
		map[string]any{"replicas": int64(1)},
	)
}

func (s *Service) Delete(ctx context.Context, name string) error {
	if _, err := s.get(ctx, name); err != nil {
		return err
	}
	return mapStoreError(s.k8s.DeleteSandbox(ctx, name), name)
}

func (s *Service) get(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	obj, err := s.k8s.GetSandbox(ctx, name)
	if err != nil {
		return nil, mapStoreError(err, name)
	}
	return obj, nil
}

func (s *Service) patchSpec(ctx context.Context, name string, first, second map[string]any) error {
	if _, err := s.k8s.PatchSandbox(ctx, name, specPatch(first)); err == nil {
		return nil
	} else if errors.Is(err, kubernetes.ErrNotFound) {
		return models.ErrMissing.Format(name)
	}
	_, err := s.k8s.PatchSandbox(ctx, name, specPatch(second))
	return mapStoreError(err, name)
}

func specPatch(spec map[string]any) []byte {
	raw, err := json.Marshal(map[string]any{"spec": spec})
	if err != nil {
		return nil
	}
	return raw
}

func mapStoreError(err error, name string) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, kubernetes.ErrAlreadyExists):
		return models.ErrAlreadyExists.Format(name)
	case errors.Is(err, kubernetes.ErrNotFound):
		return models.ErrMissing.Format(name)
	default:
		return models.WrapInternal(err)
	}
}

var _ Store = (*kubernetes.Client)(nil)
