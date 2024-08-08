package serviceaccount

import (
	"context"
	"github.com/cloudogu/k8s-registry-lib/config"
)

type SensitiveDoguConfigProvider interface {
	GetSensitiveDoguConfig(ctx context.Context, doguName string) (SensitiveDoguConfig, error)
}

type SensitiveDoguConfigSetter interface {
	Set(ctx context.Context, key, value string) error
}

type SensitiveDoguConfigGetter interface {
	Exists(ctx context.Context, key string) (bool, error)
	Get(ctx context.Context, key string) (string, error)
}

type SensitiveDoguConfigDeleter interface {
	DeleteRecursive(ctx context.Context, key string) error
}

type SensitiveDoguConfig interface {
	SensitiveDoguConfigGetter
	SensitiveDoguConfigSetter
	SensitiveDoguConfigDeleter
}

type SensitiveDoguConfigRepository interface {
	Get(ctx context.Context, name config.SimpleDoguName) (config.DoguConfig, error)
	Update(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	SaveOrMerge(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
}
