package steps

import "time"

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

// TODO Has no effect. An error will be requeued with exponential backoff by the reconciler
func RequeueAfterWithError(requeueAfter time.Duration, err error) StepResult {
	return StepResult{
		RequeueAfter: requeueAfter,
		Err:          err,
		Continue:     true,
	}
}

func RequeueWithError(err error) StepResult {
	return StepResult{
		Err:      err,
		Continue: true,
	}
}
