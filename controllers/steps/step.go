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

func RequeueWithError(err error) StepResult {
	return StepResult{
		Err:      err,
		Continue: true,
	}
}
