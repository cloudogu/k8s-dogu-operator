package serviceaccount

import "context"

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
