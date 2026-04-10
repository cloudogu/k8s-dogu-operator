package exposedport

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudogu/cesapp-lib/core"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	exposedPortsConfigMapName = "k8s-ces-gateway-config"
)

type Values struct {
	ports map[string]interface{} `yaml:"ports"`
}

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

func (epm *exposedPortsManager) readValues(ctx context.Context) (*v1.ConfigMap, *Values, error) {
	cm, err := epm.configMapInterface.Get(ctx, "k8s-ces-gateway-config", metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	var values Values
	if err := yaml.Unmarshal([]byte(cm.Data["values"]), &values); err != nil {
		return nil, nil, err
	}

	return cm, &values, err
}

func (epm *exposedPortsManager) updateConfigMap(ctx context.Context, cm *v1.ConfigMap, values *Values) (*v1.ConfigMap, error) {
	updated, err := yaml.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize values: %w", err)
	}

	cm.Data["values"] = string(updated)

	return epm.configMapInterface.Update(ctx, cm, metav1.UpdateOptions{})
}

func (epm *exposedPortsManager) AddPorts(ctx context.Context, ports []core.ExposedPort) (*v1.ConfigMap, error) {
	if len(ports) == 0 {
		return nil, nil
	}

	cm, err := epm.configMapInterface.Get(ctx, exposedPortsConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
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
		return nil, err
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
	cmConfigValues := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(data["values"]), &cmConfigValues)
	if err != nil {
		return nil, err
	}
	traefik := cmConfigValues["traefik"].(map[string]interface{})
	cmPorts := traefik["ports"].(map[string]interface{})

	for _, port := range ports {
		p := Port{
			Port:     port.Host,
			Protocol: strings.ToUpper(port.Type),
		}
		portName := fmt.Sprintf("%s-%d", strings.ToLower(p.Protocol), p.Port)

		cmPorts[portName] = p
	}

	cmBytes, err := yaml.Marshal(cmConfigValues)
	data["values"] = strings.TrimSuffix(string(cmBytes), "\n")
	return data, nil
}

func (epm *exposedPortsManager) deletePorts(data map[string]string, ports []core.ExposedPort) (map[string]string, error) {
	cmConfigValues := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(data["values"]), &cmConfigValues)
	if err != nil {
		return nil, err
	}
	traefik := cmConfigValues["traefik"].(map[string]interface{})
	cmPorts := traefik["ports"].(map[string]interface{})

	for _, port := range ports {
		portName := fmt.Sprintf("%s-%d", strings.ToLower(port.Type), port.Host)
		delete(cmPorts, portName)
	}

	cmBytes, err := yaml.Marshal(cmConfigValues)
	data["values"] = strings.TrimSuffix(string(cmBytes), "\n")
	return data, nil
}
