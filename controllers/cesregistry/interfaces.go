package cesregistry

import "github.com/cloudogu/k8s-registry-lib/dogu"

type doguVersionRegistry interface {
	dogu.DoguVersionRegistry
}

type localDoguDescriptorRepository interface {
	dogu.LocalDoguDescriptorRepository
}
