package initfx

import (
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var NewKubernetesClientSet = kubernetes.NewForConfig
var NewEcoSystemClientSet = doguClient.NewForConfig

func NewK8sClient(mgr manager.Manager) client.Client {
	return mgr.GetClient()
}

func NewDoguInterface(ecosystemClientSet doguClient.EcoSystemV2Interface, config config.OperatorConfig) doguClient.DoguInterface {
	return ecosystemClientSet.Dogus(config.Namespace)
}

func NewDoguRestartInterface(ecosystemClientSet doguClient.EcoSystemV2Interface, config config.OperatorConfig) doguClient.DoguRestartInterface {
	return ecosystemClientSet.DoguRestarts(config.Namespace)
}

func NewConfigMapInterface(clientSet kubernetes.Interface, operatorConfig config.OperatorConfig) v1.ConfigMapInterface {
	return clientSet.CoreV1().ConfigMaps(operatorConfig.Namespace)
}

func NewSecretInterface(clientSet kubernetes.Interface, operatorConfig config.OperatorConfig) v1.SecretInterface {
	return clientSet.CoreV1().Secrets(operatorConfig.Namespace)
}

func NewDeploymentInterface(clientSet kubernetes.Interface, operatorConfig config.OperatorConfig) appsv1.DeploymentInterface {
	return clientSet.AppsV1().Deployments(operatorConfig.Namespace)
}

func NewPodInterface(clientSet kubernetes.Interface, operatorConfig config.OperatorConfig) v1.PodInterface {
	return clientSet.CoreV1().Pods(operatorConfig.Namespace)
}

func NewServiceInterface(clientSet kubernetes.Interface, operatorConfig config.OperatorConfig) v1.ServiceInterface {
	return clientSet.CoreV1().Services(operatorConfig.Namespace)
}

func NewPersistentVolumeClaimInterface(clientSet kubernetes.Interface, operatorConfig config.OperatorConfig) v1.PersistentVolumeClaimInterface {
	return clientSet.CoreV1().PersistentVolumeClaims(operatorConfig.Namespace)
}

func NewEventRecorder(mgr manager.Manager) record.EventRecorder {
	return mgr.GetEventRecorderFor("k8s-dogu-operator")
}

func NewRestClient(clientSet kubernetes.Interface) rest.Interface {
	return clientSet.CoreV1().RESTClient()
}

func NewScheme(mgr manager.Manager) *runtime.Scheme {
	return mgr.GetScheme()
}
