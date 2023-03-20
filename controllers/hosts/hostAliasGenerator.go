package hosts

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"net"
	"strconv"
	"strings"

	"github.com/cloudogu/cesapp-lib/registry"
)

const (
	useInternalIPKey      = "/k8s/use_internal_ip"
	internalIPKey         = "/k8s/internal_ip"
	fqdnKey               = "/fqdn"
	additionalHostsPrefix = "/containers/additional_hosts/"
)

type config struct {
	fqdn            string
	useInternalIP   bool
	internalIP      net.IP
	additionalHosts map[string]string
}

type hostAliasGenerator struct {
	globalConfig registry.ConfigurationContext
}

// NewHostAliasGenerator creates a generator with the ability to return host aliases from the configured internal ip, additional hosts and fqdn.
func NewHostAliasGenerator(registry registry.Registry) *hostAliasGenerator {
	return &hostAliasGenerator{
		globalConfig: registry.GlobalConfig(),
	}
}

// Generate patches the given deployment with the hosts configuration provided.
func (d *hostAliasGenerator) Generate() (hostAliases []v1.HostAlias, err error) {
	hostsConfig, err := d.retrieveConfig()
	if err != nil {
		return nil, err
	}

	if hostsConfig.useInternalIP {
		splitDnsHostAlias := v1.HostAlias{
			IP:        hostsConfig.internalIP.String(),
			Hostnames: []string{hostsConfig.fqdn},
		}
		hostAliases = append(hostAliases, splitDnsHostAlias)
	}

	for hostName, ip := range hostsConfig.additionalHosts {
		addHostAlias := v1.HostAlias{
			IP:        ip,
			Hostnames: []string{hostName},
		}
		hostAliases = append(hostAliases, addHostAlias)
	}

	return hostAliases, nil
}

// retrieveConfig reads hosts-specific keys from the global configuration and creates a config object.
func (d *hostAliasGenerator) retrieveConfig() (*config, error) {
	fqdn, err := d.retrieveFQDN()
	if err != nil {
		return nil, err
	}

	hostsConfig := &config{
		fqdn: fqdn,
	}

	hostsConfig.useInternalIP, err = d.retrieveUseInternalIP()
	if err != nil {
		return nil, err
	}

	hostsConfig.internalIP, err = d.retrieveInternalIP(hostsConfig.useInternalIP)
	if err != nil {
		return nil, err
	}

	hostsConfig.additionalHosts, err = d.retrieveAdditionalHosts()
	if err != nil {
		return nil, err
	}

	return hostsConfig, nil
}

func (d *hostAliasGenerator) retrieveFQDN() (string, error) {
	fqdn, err := d.globalConfig.Get(fqdnKey)
	if err != nil {
		return "", nil
	}

	return fqdn, err
}

func (d *hostAliasGenerator) retrieveUseInternalIP() (useInternalIP bool, err error) {
	useInternalIPRaw, err := d.globalConfig.Get(useInternalIPKey)
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return false, err
	} else if err == nil {
		useInternalIP, err = parseUseInternalIP(useInternalIPRaw)
		if err != nil {
			return false, err
		}
	}

	return useInternalIP, nil
}

func (d *hostAliasGenerator) retrieveInternalIP(useInternalIP bool) (internalIP net.IP, err error) {
	internalIPRaw, err := d.globalConfig.Get(internalIPKey)
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return nil, err
	} else if err == nil {
		internalIP, err = parseInternalIP(internalIPRaw, useInternalIP)
		if err != nil {
			return nil, err
		}
	}

	return internalIP, nil
}

func (d *hostAliasGenerator) retrieveAdditionalHosts() (map[string]string, error) {
	globalConfig, err := d.globalConfig.GetAll()
	if err != nil {
		return nil, err
	}

	additionalHosts := map[string]string{}
	for key, value := range globalConfig {
		if strings.HasPrefix(key, additionalHostsPrefix) {
			hostName := strings.TrimPrefix(key, additionalHostsPrefix)
			additionalHosts[hostName] = value
		}
	}
	return additionalHosts, nil
}

func parseUseInternalIP(raw string) (bool, error) {
	if raw == "" {
		return false, nil
	}
	useInternalIP, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("failed to parse value '%s' of field 'k8s/use_internal_ip' in global config: %w", raw, err)
	}
	return useInternalIP, nil
}

func parseInternalIP(raw string, useInternalIP bool) (net.IP, error) {
	if !useInternalIP {
		return nil, nil
	}

	ip := net.ParseIP(raw)
	if ip == nil {
		return nil, fmt.Errorf("failed to parse value '%s' of field 'k8s/internal_ip' in global config: not a valid ip", raw)
	}

	return ip, nil
}
