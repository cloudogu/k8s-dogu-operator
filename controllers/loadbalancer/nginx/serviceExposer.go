package nginx

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/net"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

const (
	portHTTP  = 80
	portHTTPS = 443
)

type ingressNginxTcpUpdExposer struct {
	client client.Client
}

// NewIngressNginxTCPUDPExposer creates a new instance of the ingressNginxTcpUpdExposer.
func NewIngressNginxTCPUDPExposer(client client.Client) *ingressNginxTcpUpdExposer {
	return &ingressNginxTcpUpdExposer{client: client}
}

// ExposeOrUpdateDoguServices creates or updates the matching tcp/udp configmap for nginx routing.
// It also deletes all legacy entries from the dogu. Port 80 and 443 will be ignored.
//
// see: https://kubernetes.github.io/ingress-nginx/user-guide/exposing-tcp-udp-services/
func (intue *ingressNginxTcpUpdExposer) ExposeOrUpdateDoguServices(ctx context.Context, namespace string, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)
	if len(dogu.ExposedPorts) < 1 {
		logger.Info("Skipping tcp/udp port creation because the dogu has no exposed ports...")
		return nil
	}

	err := intue.exposeOrUpdatePortsForProtocol(ctx, namespace, dogu, net.TCP)
	if err != nil {
		return err
	}

	return intue.exposeOrUpdatePortsForProtocol(ctx, namespace, dogu, net.UDP)
}

func (intue *ingressNginxTcpUpdExposer) exposeOrUpdatePortsForProtocol(ctx context.Context, namespace string, dogu *core.Dogu, protocol net.Protocol) error {
	cm, err := intue.getNginxExposeConfigmap(ctx, namespace, protocol)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to get configmap %s: %w", getConfigMapNameForProtocol(protocol), err)
	}

	if err != nil && apierrors.IsNotFound(err) {
		_, err = intue.createNginxExposeConfigMapForProtocol(ctx, namespace, dogu, protocol)
		return err
	}

	logger := log.FromContext(ctx)
	oldLen := len(cm.Data)
	cm.Data = filterDoguServices(cm, namespace, dogu)
	doguExposedPortsByType := getExposedPortsByType(dogu, string(protocol))
	if oldLen == len(cm.Data) && len(doguExposedPortsByType) == 0 {
		logger.Info(fmt.Sprintf("Skipping %s port exposing because there are no changes...", string(protocol)))
		return nil
	}

	for _, port := range doguExposedPortsByType {
		cm.Data[getServiceEntryKey(port)] = getServiceEntryValue(namespace, dogu, port)
	}

	logger.Info(fmt.Sprintf("Update %s port exposing...", string(protocol)))
	err = intue.client.Update(ctx, cm)
	if err != nil {
		return fmt.Errorf("failed to update configmap %s: %w", cm.Name, err)
	}

	return nil
}

func (intue *ingressNginxTcpUpdExposer) getNginxExposeConfigmap(ctx context.Context, namespace string, protocol net.Protocol) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	cmName := getConfigMapNameForProtocol(protocol)
	return cm, intue.client.Get(ctx, types.NamespacedName{Name: cmName, Namespace: namespace}, cm)
}

func (intue *ingressNginxTcpUpdExposer) createNginxExposeConfigMapForProtocol(ctx context.Context, namespace string, dogu *core.Dogu, protocol net.Protocol) (*corev1.ConfigMap, error) {
	exposedPorts := getExposedPortsByType(dogu, string(protocol))
	if len(exposedPorts) < 1 {
		return nil, nil
	}

	cmName := getConfigMapNameForProtocol(protocol)
	cmData := map[string]string{}
	for _, port := range exposedPorts {
		cmData[getServiceEntryKey(port)] = getServiceEntryValue(namespace, dogu, port)
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: cmName, Namespace: namespace},
		Data:       cmData,
	}

	err := intue.client.Create(ctx, cm)
	if err != nil {
		return nil, fmt.Errorf("failed to create configmap %s: %w", cmName, err)
	}

	return cm, nil
}

func getConfigMapNameForProtocol(protocol net.Protocol) string {
	return fmt.Sprintf("%s-services", strings.ToLower(string(protocol)))
}

func getServiceEntryKey(port core.ExposedPort) string {
	return fmt.Sprintf("%d", port.Host)
}

func getServiceEntryValue(namespace string, dogu *core.Dogu, port core.ExposedPort) string {
	return fmt.Sprintf("%s:%d", getServiceEntryValuePrefix(namespace, dogu), port.Container)
}

func getServiceEntryValuePrefix(namespace string, dogu *core.Dogu) string {
	return fmt.Sprintf("%s/%s", namespace, dogu.GetSimpleName())
}

func filterDoguServices(cm *corev1.ConfigMap, namespace string, dogu *core.Dogu) map[string]string {
	data := cm.Data
	if data == nil {
		return map[string]string{}
	}

	for key, value := range data {
		if strings.Contains(value, getServiceEntryValuePrefix(namespace, dogu)) {
			delete(data, key)
		}
	}

	return data
}

func getExposedPortsByType(dogu *core.Dogu, protocol string) []core.ExposedPort {
	var result []core.ExposedPort
	for _, port := range dogu.ExposedPorts {
		if port.Host == portHTTP || port.Host == portHTTPS {
			continue
		}

		if strings.EqualFold(port.Type, protocol) {
			result = append(result, port)
		}
	}

	return result
}

// DeleteDoguServices removes all Dogu related entries in the corresponding tcp/udp configmaps.
// If the configmap has no entries left this method won't delete the configmap. This would lead to numerous
// errors in the nginx log.
func (intue *ingressNginxTcpUpdExposer) DeleteDoguServices(ctx context.Context, namespace string, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)
	if len(dogu.ExposedPorts) < 1 {
		logger.Info("Skipping tcp/udp port deletion because the dogu has no exposed ports...")
		return nil
	}

	err := intue.deletePortsForProtocol(ctx, namespace, dogu, net.TCP)
	if err != nil {
		return err
	}

	return intue.deletePortsForProtocol(ctx, namespace, dogu, net.UDP)
}

func (intue *ingressNginxTcpUpdExposer) deletePortsForProtocol(ctx context.Context, namespace string, dogu *core.Dogu, protocol net.Protocol) error {
	cm, err := intue.getNginxExposeConfigmap(ctx, namespace, protocol)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get configmap %s: %w", getConfigMapNameForProtocol(protocol), err)
		} else {
			return nil
		}
	}

	if cm.Data == nil || len(cm.Data) == 0 {
		return nil
	}

	for key, value := range cm.Data {
		if strings.Contains(value, getServiceEntryValuePrefix(namespace, dogu)) {
			delete(cm.Data, key)
		}
	}

	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("Update %s port exposing...", string(protocol)))
	// Do not delete the configmap, even it contains no ports. That would throw errors in nginx-ingress log.
	return intue.client.Update(ctx, cm)
}
