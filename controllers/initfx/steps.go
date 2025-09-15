package initfx

import (
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/usecase"
	"go.uber.org/fx"
)

func AsDeleteStep(f any, additionalAnnotations ...fx.Annotation) any {
	return fx.Annotate(
		f,
		append(additionalAnnotations,
			fx.As(new(usecase.Step)),
			fx.ResultTags(`group:"deleteSteps"`),
		)...,
	)
}

func AsInstallOrChangeStep(f any, additionalAnnotations ...fx.Annotation) any {
	return fx.Annotate(
		f,
		append(additionalAnnotations,
			fx.As(new(usecase.Step)),
			fx.ResultTags(`group:"installOrChangeSteps"`),
		)...,
	)
}
