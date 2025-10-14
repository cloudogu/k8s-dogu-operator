package steps

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

type Step interface {
	Run(ctx context.Context, resource *v2.Dogu) StepResult
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
