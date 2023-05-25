package resource

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const cesLoadbalancerName = "ces-loadbalancer"

type IngressNginxTcpUpdExposer struct {
	client client.Client
}

func NewIngressNginxTCPUDPExposer(client client.Client) *IngressNginxTcpUpdExposer {
	return &IngressNginxTcpUpdExposer{client: client}
}

type tcpUpdServiceExposer interface {
	exposeService(port core.ExposedPort) error
}

type doguExposedPortHandler struct {
	client client.Client
}

// NewDoguExposedPortHandler creates a new instance of doguExposedPortHandler.
func NewDoguExposedPortHandler(client client.Client) *doguExposedPortHandler {
	return &doguExposedPortHandler{client: client}
}

// CreateOrUpdateCesLoadbalancerService updates the loadbalancer service "ces-loadbalancer" with the dogu exposed ports.
// If the service is not existent in cluster, it will be created.
// If the dogu has no exposed ports, this method return an empty service object and nil.
func (deph *doguExposedPortHandler) CreateOrUpdateCesLoadbalancerService(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) (*corev1.Service, error) {
	if len(dogu.ExposedPorts) == 0 {
		return &corev1.Service{}, nil
	}

	exposedService, err := deph.getCesLoadBalancerService(ctx, doguResource)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get service %s: %w", cesLoadbalancerName, err)
	}

	if err != nil && apierrors.IsNotFound(err) {
		createLoadbalancerService, createErr := deph.createCesLoadbalancerService(ctx, doguResource, dogu)
		if createErr != nil {
			return nil, fmt.Errorf("failed to create %s service: %w", cesLoadbalancerName, createErr)
		}
		// TODO update tcp/udp configmaps
		return createLoadbalancerService, nil
	}

	exposedService = updateCesLoadbalancerService(dogu, exposedService)
	// TODO update tcp/udp configmaps

	return exposedService, deph.updateService(ctx, exposedService)
}

func (deph *doguExposedPortHandler) getCesLoadBalancerService(ctx context.Context, doguResource *k8sv1.Dogu) (*corev1.Service, error) {
	exposedService := &corev1.Service{}
	cesLoadBalancerService := types.NamespacedName{Name: cesLoadbalancerName, Namespace: doguResource.Namespace}

	return exposedService, deph.client.Get(ctx, cesLoadBalancerService, exposedService)
}

func (deph *doguExposedPortHandler) createCesLoadbalancerService(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) (*corev1.Service, error) {
	ipSingleStackPolicy := corev1.IPFamilyPolicySingleStack
	exposedService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cesLoadbalancerName,
			Namespace: doguResource.Namespace,
			Labels:    GetAppLabel(),
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeLoadBalancer,
			IPFamilyPolicy: &ipSingleStackPolicy,
			IPFamilies:     []corev1.IPFamily{corev1.IPv4Protocol},
			Selector:       doguResource.GetDoguNameLabel(),
		},
	}

	var servicePorts []corev1.ServicePort
	for _, exposedPort := range dogu.ExposedPorts {
		servicePorts = append(servicePorts, getServicePortFromExposedPort(dogu, exposedPort))
	}
	exposedService.Spec.Ports = servicePorts

	err := deph.client.Create(ctx, exposedService)
	if err != nil {
		return exposedService, fmt.Errorf("failed to create %s service: %w", cesLoadbalancerName, err)
	}

	return exposedService, nil
}

func getServicePortFromExposedPort(dogu *core.Dogu, exposedPort core.ExposedPort) corev1.ServicePort {
	return corev1.ServicePort{
		Name:       fmt.Sprintf("%s%d", getServicePortNamePrefix(dogu), exposedPort.Host),
		Protocol:   corev1.Protocol(strings.ToUpper(exposedPort.Type)),
		Port:       int32(exposedPort.Host),
		TargetPort: intstr.FromInt(exposedPort.Container),
	}
}

func getServicePortNamePrefix(dogu *core.Dogu) string {
	return fmt.Sprintf("%s-", dogu.GetSimpleName())
}

func updateCesLoadbalancerService(dogu *core.Dogu, service *corev1.Service) *corev1.Service {
	service.Spec.Ports = filterDoguServicePorts(dogu, service)

	for _, exposedPort := range dogu.ExposedPorts {
		service.Spec.Ports = append(service.Spec.Ports, getServicePortFromExposedPort(dogu, exposedPort))
	}

	return service
}

func filterDoguServicePorts(dogu *core.Dogu, service *corev1.Service) []corev1.ServicePort {
	var doguServicePorts []corev1.ServicePort

	for _, servicePort := range service.Spec.Ports {
		servicePortName := servicePort.Name
		doguPrefix := getServicePortNamePrefix(dogu)
		f := strings.HasPrefix(servicePortName, doguPrefix)
		if !f {
			doguServicePorts = append(doguServicePorts, servicePort)
		}
	}

	return doguServicePorts
}

func (deph *doguExposedPortHandler) updateService(ctx context.Context, exposedService *corev1.Service) error {
	err := deph.client.Update(ctx, exposedService)
	if err != nil {
		return fmt.Errorf("failed to update %s service: %w", cesLoadbalancerName, err)
	}
	return nil
}

// RemoveExposedPorts removes given dogu exposed ports from the loadbalancer service.
// If these ports are the only ones, the service will be deleted.
// If the dogu has no exposed ports, the method returns nil.
func (deph *doguExposedPortHandler) RemoveExposedPorts(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error {
	if len(dogu.ExposedPorts) == 0 {
		return nil
	}

	exposedService, err := deph.getCesLoadBalancerService(ctx, doguResource)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get service %s: %w", cesLoadbalancerName, err)
		} else {
			return nil
		}
	}

	ports := filterDoguServicePorts(dogu, exposedService)
	if len(ports) > 0 {
		exposedService.Spec.Ports = ports
		return deph.updateService(ctx, exposedService)
	}

	err = deph.client.Delete(ctx, exposedService)
	if err != nil {
		return fmt.Errorf("failed to delete service %s: %w", cesLoadbalancerName, err)
	}

	return nil
}
