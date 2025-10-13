package security

import (
	"errors"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

type validator struct {
}

type Validator interface {
	ValidateSecurity(doguDescriptor *core.Dogu, doguResource *k8sv2.Dogu) error
}

// NewValidator constructs a new *security.Validator.
func NewValidator() Validator {
	return &validator{}
}

// ValidateSecurity verifies the security fields of dogu descriptor and resource for correctness.
func (v *validator) ValidateSecurity(doguDescriptor *core.Dogu, doguResource *k8sv2.Dogu) error {
	descriptorErr := doguDescriptor.ValidateSecurity()
	if descriptorErr != nil {
		descriptorErr = fmt.Errorf("invalid security field in dogu descriptor: %w", descriptorErr)
	}

	resourceErr := doguResource.ValidateSecurity()
	if resourceErr != nil {
		resourceErr = fmt.Errorf("invalid security field in dogu resource: %w", resourceErr)
	}

	if descriptorErr != nil || resourceErr != nil {
		return errors.Join(descriptorErr, resourceErr)
	}

	return nil
}
