package controllers

import (
	"bytes"
	"context"
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"time"
)

const doguCustomK8sResourceDirectory = "/k8s/"

type podFileExtractor struct {
	k8sClient client.Client
	config    *rest.Config
	clientSet *kubernetes.Clientset
}

func newPodFileExtractor(k8sClient client.Client, restConfig *rest.Config, clientSet *kubernetes.Clientset) *podFileExtractor {
	return &podFileExtractor{k8sClient: k8sClient, config: restConfig, clientSet: clientSet}
}

// extractK8sResourcesFromContainer enumerates K8s resources and returns them in a map file->content. The map will be
// empty if there are no files.
func (fe *podFileExtractor) extractK8sResourcesFromContainer(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) (map[string]string, error) {
	logger := log.FromContext(ctx)
	currentNamespace := doguResource.ObjectMeta.Namespace

	podspec, containerPodName := createExecPodSpec(currentNamespace, dogu)
	logger.Info("Creating new exec pod " + containerPodName)
	err := fe.k8sClient.Create(ctx, &podspec)
	if err != nil {
		return nil, fmt.Errorf("could not create pod for file extraction: %w", err)
	}
	defer func() {
		logger.Info("Cleaning up intermediate exec pod for dogu ", dogu.Name)
		err = fe.k8sClient.Delete(ctx, &podspec)
		if err != nil {
			logger.Error(fmt.Errorf("failed to delete custom dogu descriptor: %w", err), "Error while deleting intermediate ")
		}
	}()

	podExecKey := createPodExecObjectKey(currentNamespace, containerPodName)

	err = fe.findPod(ctx, podExecKey, logger, containerPodName)
	if err != nil {
		return nil, err
	}

	podexec, err := newPodExec(fe.config, fe.clientSet, currentNamespace, containerPodName)

	out, _, err := podexec.execCmd([]string{"/bin/ls", "/k8s/"})
	if err != nil {
		return nil, fmt.Errorf("could not enumerate K8s resources in execPod %s: %w", containerPodName, err)
	}

	resultDocs := make(map[string]string)
	if out.Len() == 0 {
		logger.Info("No custom K8s resource files found.")
		return resultDocs, nil
	}

	for _, file := range strings.Split(out.String(), " ") {
		trimmedFile := doguCustomK8sResourceDirectory + strings.TrimSpace(file)
		logger.Info("Reading k8s resource " + trimmedFile)

		out, _, err = podexec.execCmd([]string{"/bin/cat", trimmedFile})
		if err != nil {
			return nil, fmt.Errorf("could not enumerate K8s resources in execPod %s: %w", containerPodName, err)
		}

		resultDocs[trimmedFile] = out.String()
	}

	return resultDocs, nil
}

func (fe *podFileExtractor) findPod(ctx context.Context, podExecKey client.ObjectKey, logger logr.Logger, containerPodName string) error {
	lePod := corev1.Pod{}
	const maxTries = 10

	for i := 0; i < maxTries; i++ {
		if i >= maxTries-1 {
			return fmt.Errorf("quitting dogu installation because exec pod %s could not be found", containerPodName)
		}

		err := fe.k8sClient.Get(ctx, podExecKey, &lePod)
		if err != nil {
			logger.Error(err, "Error while finding exec pod "+containerPodName+". Will try again.")
			sleep(logger, i)
			continue
		}

		leStatus := lePod.Status.Phase
		switch leStatus {
		case corev1.PodRunning:
			logger.Info("Found a ready exec pod " + containerPodName)
			return nil
		case corev1.PodFailed, corev1.PodSucceeded:
			return fmt.Errorf("quitting dogu installation because exec pod %s failed with status %s or did not come up in time", containerPodName, leStatus)
		default:
			logger.Info(fmt.Sprintf("Found exec pod %s but with status phase %+v", containerPodName, leStatus))
			sleep(logger, i)
			continue
		}
	}

	return fmt.Errorf("unexpected loop end while finding exec pod %s", containerPodName)
}

func sleep(logger logr.Logger, sleepIntervalInSec int) {
	logger.Info(fmt.Sprintf("Exec pod not found. Trying again in %d second(s)", sleepIntervalInSec))
	time.Sleep(time.Duration(sleepIntervalInSec) * time.Second) // linear rising backoff
}

func createPodExecObjectKey(k8sNamespace, containerPodName string) client.ObjectKey {
	return client.ObjectKey{
		Namespace: k8sNamespace,
		Name:      containerPodName,
	}
}

func createExecPodSpec(k8sNamespace string, dogu *core.Dogu) (corev1.Pod, string) {
	containerName := dogu.GetSimpleName() + "-execpod"
	image := dogu.Image + ":" + dogu.Version
	// command is of no importance because the pod will be killed after success
	doNothingCommand := []string{"/bin/sleep", "60"}
	labels := map[string]string{"app": "ces", "dogu": containerName}

	return corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:        containerName,
			Namespace:   k8sNamespace,
			Labels:      labels,
			Annotations: make(map[string]string),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            containerName,
					Image:           image,
					Command:         doNothingCommand,
					ImagePullPolicy: corev1.PullIfNotPresent,
				},
			},
		},
	}, containerName
}

// podExec executes commands in a running K8s container
type podExec struct {
	restConfig *rest.Config
	*kubernetes.Clientset
	namespace     string
	podName       string
	containerName string
}

func newPodExec(config *rest.Config, clientSet *kubernetes.Clientset, namespace, containerPodName string) (*podExec, error) {
	config.APIPath = "/api"
	config.GroupVersion = &schema.GroupVersion{Version: "v1"}
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}

	return &podExec{
		restConfig:    config,
		Clientset:     clientSet,
		namespace:     namespace,
		podName:       containerPodName,
		containerName: containerPodName,
	}, nil
}

// execCmd executes arbitrary commands in a pod container.
func (p *podExec) execCmd(command []string) (out *bytes.Buffer, errOut *bytes.Buffer, err error) {
	in := &bytes.Buffer{}
	out = &bytes.Buffer{}
	errOut = &bytes.Buffer{}

	ioStreams := genericclioptions.IOStreams{
		In:     in,
		Out:    out,
		ErrOut: errOut,
	}

	options := &exec.ExecOptions{
		StreamOptions: exec.StreamOptions{
			Namespace:       p.namespace,
			PodName:         p.podName,
			ContainerName:   p.containerName,
			Stdin:           true,
			TTY:             false,
			Quiet:           false,
			InterruptParent: nil,
			IOStreams:       ioStreams,
		},
		Command:       command,
		Executor:      &exec.DefaultRemoteExecutor{},
		PodClient:     p.Clientset.CoreV1(),
		GetPodTimeout: 0,
		Config:        p.restConfig,
	}

	err = options.Run()
	if err != nil {
		return nil, nil, fmt.Errorf("could not run exec operation: %w", err)
	}

	return out, errOut, nil
}
