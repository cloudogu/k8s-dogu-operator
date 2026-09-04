package v3

import (
	"context"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
)

type Step interface {
	Run(ctx context.Context, resource *v3beta1.Dogu) StepResult
}

type StepResult struct {
	RequeueAfter time.Duration
	Err          error
	Continue     bool
}

func RequeueAfter(requeueAfter time.Duration) StepResult {
	return StepResult{
		RequeueAfter: requeueAfter,
	}
}

func Continue() StepResult {
	return StepResult{
		Continue: true,
	}
}

func Abort() StepResult {
	return StepResult{
		Continue: false,
	}
}

func RequeueWithError(err error) StepResult {
	return StepResult{
		Err: err,
	}
}
