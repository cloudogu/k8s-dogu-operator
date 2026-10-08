package initfx

import (
	doguClient "github.com/cloudogu/k8s-dogu-lib/v3/client"
	"github.com/cloudogu/k8s-dogu-lib/v3/client/typed/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
)

func NewDoguV3Interface(doguClientset doguClient.Interface, config *config.OperatorConfig) v3beta1.DoguInterface {
	return doguClientset.DoguV3beta1().Dogus(config.Namespace)
}
