package serviceaccount

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	"github.com/cloudogu/cesapp/v4/keys"
	"github.com/cloudogu/cesapp/v4/registry"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/pkg/errors"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"net/url"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

// DoguRegistry is used to fetch the dogu descriptor
type DoguRegistry interface {
	GetDogu(*k8sv1.Dogu) (*core.Dogu, error)
}

// Creator is the unit to handle the creation of service accounts
type Creator struct {
	Client                 kubernetes.Interface `json:"client"`
	CoreV1RestClient       rest.Interface       `json:"coreV1RestClient"`
	Registry               registry.Registry    `json:"registry"`
	CommandExecutorCreator func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error)
}

// NewServiceAccountCreator creates a new instance of ServiceAccountCreator
func NewServiceAccountCreator(client kubernetes.Interface, coreV1RestClient rest.Interface, registry registry.Registry) *Creator {
	return &Creator{
		Client:                 client,
		CoreV1RestClient:       coreV1RestClient,
		Registry:               registry,
		CommandExecutorCreator: remotecommand.NewSPDYExecutor,
	}
}

// CreateServiceAccounts creates all service account for a given dogu
func (c *Creator) CreateServiceAccounts(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	for _, serviceAccount := range dogu.ServiceAccounts {
		parentPath := "sa-" + serviceAccount.Type
		doguConfig := c.Registry.DoguConfig(dogu.GetSimpleName())

		// check if the service account already exists
		exists, err := c.serviceAccountExists(parentPath, doguConfig)
		if err != nil {
			return fmt.Errorf("failed to check existence of service account: %w", err)
		}

		if exists {
			logger.Info("skipping creation of service account because it already exists")
			continue
		}

		// check if the dogu is enabled
		doguRegistry := c.Registry.DoguRegistry()
		enabled, err := doguRegistry.IsEnabled(serviceAccount.Type)
		if err != nil {
			return fmt.Errorf("failed to check if dogu %s is enabled: %w", serviceAccount.Type, err)
		}

		// check if the service account is optional
		if !enabled && c.isOptionalServiceAccount(dogu, serviceAccount.Type) {
			logger.Info("skipping optional service account creation for %s, because the dogu is not installed", serviceAccount.Type)
			continue
		}

		// check if the service account dogu is enabled and mandatory
		if !enabled && !c.isOptionalServiceAccount(dogu, serviceAccount.Type) {
			return fmt.Errorf("service account dogu is not enabled and not optional")
		}

		// get the service account dogu.json
		targetDescriptor, err := doguRegistry.Get(serviceAccount.Type)
		if err != nil {
			return fmt.Errorf("failed to get service account dogu.json: %w", err)
		}

		//get the create command
		createCommand := c.getCreateCommand(targetDescriptor)
		if createCommand == nil {
			return fmt.Errorf("service account dogu does not expose create command")
		}

		pod, err := c.getPodForServiceAccount(ctx, doguResource, serviceAccount)
		if err != nil {
			return fmt.Errorf("failed to get pod for sa dogu %s: %w", serviceAccount.Type, err)
		}

		// create request
		saParams := append(serviceAccount.Params, dogu.GetSimpleName())
		req := c.getCreateExecRequest(pod, doguResource, createCommand, saParams)

		// execute request
		exec, err := c.CommandExecutorCreator(ctrl.GetConfigOrDie(), "POST", req.URL())
		if err != nil {
			return fmt.Errorf("failed to create new spdy executor: %w", err)
		}

		buffer := bytes.NewBuffer([]byte{})
		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:  os.Stdin,
			Stdout: buffer,
			Stderr: os.Stderr,
			Tty:    true,
		})

		if err != nil {
			return fmt.Errorf("failed to exec stream: %w", err)
		}

		// parse service accounts
		saCreds, err := c.parseServiceCommandOutput(buffer)
		if err != nil {
			return fmt.Errorf("failed to parse service account: %w", err)
		}

		// get dogu public key
		publicKey, err := c.getPublicKey(doguConfig)
		if err != nil {
			return fmt.Errorf("failed to read public key from string: %w", err)
		}

		// write credentials
		err = c.writeServiceAccounts(doguConfig, serviceAccount, saCreds, publicKey)
		if err != nil {
			return fmt.Errorf("failed to write service account: %w", err)
		}
	}

	return nil
}

func (c *Creator) getPublicKey(doguConfig registry.ConfigurationContext) (*keys.PublicKey, error) {
	keyProviderStr, err := c.Registry.GlobalConfig().Get("key_provider")
	if err != nil {
		return nil, fmt.Errorf("failed to get key provider: %w", err)
	}
	keyProvider, err := keys.NewKeyProvider(core.Keys{Type: keyProviderStr})
	if err != nil {
		return nil, fmt.Errorf("failed to create keyprovider: %w", err)
	}
	pubKeyStr, err := doguConfig.Get(registry.KeyDoguPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu public key: %w", err)
	}
	publicKey, err := keyProvider.ReadPublicKeyFromString(pubKeyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key from string: %w", err)
	}

	return publicKey, nil
}

func (c *Creator) writeServiceAccounts(doguConfig registry.ConfigurationContext, serviceAccount core.ServiceAccount, saCreds map[string]string, publicKey *keys.PublicKey) error {
	for key, value := range saCreds {
		path := "/sa-" + serviceAccount.Type + "/" + key

		encrypted, err := publicKey.Encrypt(value)
		if err != nil {
			return fmt.Errorf("failed to encrypt service account value of key %s: %w", key, err)
		}

		err = doguConfig.Set(path, encrypted)
		if err != nil {
			return fmt.Errorf("failed to set encrypted sa value of key %s: %w", key, err)
		}
	}

	return nil
}

func (c *Creator) getCreateExecRequest(pod *corev1.Pod, doguResource *k8sv1.Dogu,
	createCommand *core.ExposedCommand, params []string) *rest.Request {
	return c.CoreV1RestClient.Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(doguResource.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: append([]string{createCommand.Command}, params...),
			Stdin:   true,
			Stdout:  true,
			Stderr:  true,
			TTY:     true,
		}, scheme.ParameterCodec)
}

func (c *Creator) getPodForServiceAccount(ctx context.Context, doguResource *k8sv1.Dogu,
	serviceAccount core.ServiceAccount) (*corev1.Pod, error) {
	listOptions := metav1.ListOptions{LabelSelector: "dogu=" + serviceAccount.Type}
	pods, err := c.Client.CoreV1().Pods(doguResource.Namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("found no pods for name %s", serviceAccount.Type)
	}

	return &pods.Items[0], nil
}

func (c *Creator) getCreateCommand(dogu *core.Dogu) *core.ExposedCommand {
	var createCmd *core.ExposedCommand
	for _, cmd := range dogu.ExposedCommands {
		if cmd.Name == "service-account-create" {
			createCmd = &core.ExposedCommand{
				Name:        cmd.Name,
				Description: cmd.Description,
				Command:     cmd.Command,
			}
			break
		}
	}

	return createCmd
}

func (c *Creator) isOptionalServiceAccount(dogu *core.Dogu, sa string) bool {
	if c.containsDependency(dogu.Dependencies, sa) {
		return false
	} else if c.containsDependency(dogu.OptionalDependencies, sa) {
		return true
	}
	return false
}

func (c *Creator) containsDependency(slice []core.Dependency, dependencyName string) bool {
	if slice == nil {
		return false
	}
	for _, s := range slice {
		if s.Name == dependencyName {
			return true
		}
	}
	return false
}

func (c *Creator) serviceAccountExists(saPath string, doguConfig registry.ConfigurationContext) (bool, error) {
	exists, err := doguConfig.Exists(saPath)
	if err != nil {
		return false, errors.Wrap(err, "failed to check if service account already exists")
	}
	if exists {
		return true, nil
	}
	return false, nil
}

func (c *Creator) parseServiceCommandOutput(output io.Reader) (map[string]string, error) {
	serviceAccountSettings := make(map[string]string)

	reader := bufio.NewReader(output)
	var line []byte
	var err error
	for err == nil {
		line, _, err = reader.ReadLine()
		if err != nil && err != io.EOF {
			return nil, err
		}

		// convert to string, trim and split by :
		lineAsString := strings.TrimSpace(string(line))
		if len(lineAsString) > 0 {
			kvs := strings.Split(lineAsString, ":")
			if len(kvs) != 2 {
				return nil, fmt.Errorf("invalid output from service account command on dogu")
			}

			serviceAccountSettings[strings.TrimSpace(kvs[0])] = strings.TrimSpace(kvs[1])
		}
	}

	return serviceAccountSettings, nil
}
