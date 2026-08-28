package kubernetes

import (
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrCRDMissing    = errors.New("Sandbox CRD not installed (agents.x-k8s.io/v1beta1)")
)

func Translate(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case isCRDMissing(err):
		return fmt.Errorf("%w: %w", ErrCRDMissing, err)
	case apierrors.IsAlreadyExists(err):
		return fmt.Errorf("%w: %w", ErrAlreadyExists, err)
	case apierrors.IsNotFound(err):
		return fmt.Errorf("%w: %w", ErrNotFound, err)
	default:
		return err
	}
}

func isCRDMissing(err error) bool {
	if err == nil {
		return false
	}
	if meta.IsNoMatchError(err) {
		return true
	}
	if !apierrors.IsNotFound(err) {
		return false
	}
	var statusErr *apierrors.StatusError
	if !errors.As(err, &statusErr) {
		return false
	}
	details := statusErr.ErrStatus.Details
	return details == nil || details.Name == ""
}
