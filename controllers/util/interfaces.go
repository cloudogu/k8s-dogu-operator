package util

import (
	"context"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-registry-lib/dogu"
)

type doguDescriptorRepo interface {
	Get(context.Context, dogu.DoguVersion) (*core.Dogu, error)
	GetAll(context.Context, []dogu.DoguVersion) (map[dogu.DoguVersion]*core.Dogu, error)
	Add(context.Context, dogu.SimpleDoguName, *core.Dogu) error
	DeleteAll(context.Context, dogu.SimpleDoguName) error
}
