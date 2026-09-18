package initfx

import (
	authRegClientV1 "github.com/cloudogu/k8s-auth-registration-lib/client/typed/api/v1"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v3/client"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/client/typed/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	expClientV1 "github.com/cloudogu/k8s-exposition-lib/client/typed/api/v1"
	warpClientV1 "github.com/cloudogu/k8s-warp-menu-entry-lib/client/typed/api/v1"
	"k8s.io/client-go/rest"
)

var NewAuthRegistrationClientSet = newAuthRegistrationClientSet

func newAuthRegistrationClientSet(config *rest.Config) (authRegClientV1.ApiV1Interface, error) {
	return authRegClientV1.NewForConfig(config)
}

var NewExpositionClientSet = newExpositionClientSet

func newExpositionClientSet(config *rest.Config) (expClientV1.ApiV1Interface, error) {
	return expClientV1.NewForConfig(config)
}

var NewWarpMenuEntryClientSet = newWarpMenuEntryClientSet

func newWarpMenuEntryClientSet(config *rest.Config) (warpClientV1.ApiV1Interface, error) {
	return warpClientV1.NewForConfig(config)
}

func NewDoguInterface(doguClientset doguClient.Interface, config *config.OperatorConfig) doguv2.DoguInterface {
	return doguClientset.DoguV2().Dogus(config.Namespace)
}

func NewDoguRestartInterface(doguClientset doguClient.Interface, config *config.OperatorConfig) doguv2.DoguRestartInterface {
	return doguClientset.DoguV2().DoguRestarts(config.Namespace)
}

func NewAuthRegistrationInterface(authRegistrationClientSet authRegClientV1.ApiV1Interface, operatorConfig *config.OperatorConfig) authRegClientV1.AuthRegistrationInterface {
	return authRegistrationClientSet.AuthRegistrations(operatorConfig.Namespace)
}

func NewExpositionInterface(expositionClientSet expClientV1.ApiV1Interface, operatorConfig *config.OperatorConfig) expClientV1.ExpositionInterface {
	return expositionClientSet.Expositions(operatorConfig.Namespace)
}

func NewWarpMenuEntryInterface(warpMenuEntryClientSet warpClientV1.ApiV1Interface, operatorConfig *config.OperatorConfig) warpClientV1.WarpMenuEntryInterface {
	return warpMenuEntryClientSet.WarpMenuEntries(operatorConfig.Namespace)
}
