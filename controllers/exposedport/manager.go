package exposedport

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudogu/cesapp-lib/core"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	exposedPortsConfigMapName        = "k8s-ces-gateway-config"
	initialExposedPortsConfigMapName = "initial-exposed-ports-config"
	componentOperatorReconcileLabel  = "k8s.cloudogu.com/component.config"
	k8sCesGatewayName                = "k8s-ces-gateway"
)

type Port struct {
	Port     int    `yaml:"port"`
	Protocol string `yaml:"protocol"`
}

type exposedPortsManager struct {
	configMapInterface configMapInterface
}

func NewExposedPortsManager(
	configMapInterface configMapInterface,
) ExposedPortsManager {
	return &exposedPortsManager{
		configMapInterface: configMapInterface,
	}
}

func (epm *exposedPortsManager) AddPorts(ctx context.Context, ports []core.ExposedPort) (*v1.ConfigMap, error) {
	if len(ports) == 0 {
		return nil, nil
	}

	cm, err := epm.configMapInterface.Get(ctx, exposedPortsConfigMapName, metav1.GetOptions{})
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return nil, err
		}
		cm, err = epm.createExposedPortsConfigMap(ctx)
		if err != nil {
			return nil, err
		}
	}

	data, err := epm.addPorts(cm.Data, ports)
	if err != nil {
		return nil, err
	}
	cm.Data = data

	cm, err = epm.configMapInterface.Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return cm, nil
}

func (epm *exposedPortsManager) DeletePorts(ctx context.Context, ports []core.ExposedPort) (*v1.ConfigMap, error) {
	if len(ports) == 0 {
		return nil, nil
	}

	cm, err := epm.configMapInterface.Get(ctx, exposedPortsConfigMapName, metav1.GetOptions{})
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return nil, err
		}
		cm, err = epm.createExposedPortsConfigMap(ctx)
		if err != nil {
			return nil, err
		}
	}

	data, err := epm.deletePorts(cm.Data, ports)
	if err != nil {
		return nil, err
	}
	cm.Data = data

	cm, err = epm.configMapInterface.Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return cm, nil
}

func (epm *exposedPortsManager) addPorts(data map[string]string, ports []core.ExposedPort) (map[string]string, error) {
	cmPorts, cmConfigValues, err := epm.getPortsOutOfMap(data)
	if err != nil {
		return nil, err
	}

	for _, port := range ports {
		p := Port{
			Port:     port.Host,
			Protocol: strings.ToUpper(port.Type),
		}
		portName := fmt.Sprintf("%s-%d", strings.ToLower(p.Protocol), p.Port)

		cmPorts[portName] = map[string]interface{}{
			"port":     p.Port,
			"protocol": p.Protocol,
		}
	}

	cmBytes, err := yaml.Marshal(cmConfigValues)
	if err != nil {
		return nil, err
	}
	data["values"] = strings.TrimSuffix(string(cmBytes), "\n")
	return data, nil
}

func (epm *exposedPortsManager) deletePorts(data map[string]string, ports []core.ExposedPort) (map[string]string, error) {
	cmPorts, cmConfigValues, err := epm.getPortsOutOfMap(data)
	if err != nil {
		return nil, err
	}

	for _, port := range ports {
		portName := fmt.Sprintf("%s-%d", strings.ToLower(port.Type), port.Host)
		delete(cmPorts, portName)
	}

	cmBytes, err := yaml.Marshal(cmConfigValues)
	if err != nil {
		return nil, err
	}
	data["values"] = strings.TrimSuffix(string(cmBytes), "\n")
	return data, nil
}

func (epm *exposedPortsManager) createExposedPortsConfigMap(ctx context.Context) (*v1.ConfigMap, error) {
	initialCm, err := epm.configMapInterface.Get(ctx, initialExposedPortsConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	cm := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: exposedPortsConfigMapName,
			Labels: map[string]string{
				componentOperatorReconcileLabel: k8sCesGatewayName,
			},
			Namespace: initialCm.Namespace,
		},
		Data: initialCm.Data,
	}
	createdCm, err := epm.configMapInterface.Create(ctx, cm, metav1.CreateOptions{})
	return createdCm, err
}

func (epm *exposedPortsManager) getPortsOutOfMap(data map[string]string) (map[string]interface{}, map[string]interface{}, error) {
	cmConfigValues := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(data["values"]), &cmConfigValues)
	if err != nil {
		return nil, nil, err
	}
	traefik, ok := cmConfigValues["traefik"].(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("type assertion for traefik failed")
	}
	cmPorts, ok := traefik["ports"].(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("type assertion for ports failed")
	}

	// normalize port and protocol to lowercase
	for name, raw := range cmPorts {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		if val, exists := entry["Port"]; exists {
			entry["port"] = val
			delete(entry, "Port")
		}
		if val, exists := entry["Protocol"]; exists {
			entry["protocol"] = val
			delete(entry, "Protocol")
		}

		cmPorts[name] = entry
	}

	return cmPorts, cmConfigValues, nil
}
