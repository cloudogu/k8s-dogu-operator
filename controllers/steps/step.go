package steps

import "time"

type StepResult struct {
	RequeueAfter time.Duration
	Err          error
	Continue     bool
}

func NewStepResult(requeueAfter time.Duration, err error, cont bool) StepResult {
	return StepResult{
		RequeueAfter: requeueAfter,
		Err:          err,
		Continue:     cont,
	}
}

func NewStepResultContinueIsTrue(requeueAfter time.Duration, err error) StepResult {
	return StepResult{
		RequeueAfter: requeueAfter,
		Err:          err,
		Continue:     true,
	}
}

func NewStepResultContinueIsTrueAndRequeueIsZero(err error) StepResult {
	return StepResult{
		RequeueAfter: 0,
		Err:          err,
		Continue:     true,
	}
}
