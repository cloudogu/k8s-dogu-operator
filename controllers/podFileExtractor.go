package controllers

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/cloudogu/k8s-dogu-operator/controllers/util"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const doguCustomK8sResourceDirectory = "/k8s/"

type podFileExtractor struct {
	k8sClient   client.Client
	config      *rest.Config
	clientSet   kubernetes.Interface
	podExecutor podExecutor
}

type podExecutor interface {
	exec(podExecKey *client.ObjectKey, cmdArgs ...string) (stdOut string, err error)
}

func newPodFileExtractor(k8sClient client.Client, restConfig *rest.Config, clientSet kubernetes.Interface) *podFileExtractor {
	return &podFileExtractor{
		k8sClient:   k8sClient,
		config:      restConfig,
		clientSet:   clientSet,
		podExecutor: &defaultPodExecutor{config: restConfig, clientset: clientSet},
	}
}

// ExtractK8sResourcesFromContainer enumerates K8s resources and returns them in a map filename->content. The map will be
// empty if there are no files.
func (fe *podFileExtractor) ExtractK8sResourcesFromContainer(ctx context.Context, k8sExecPod util.ExecPod) (map[string]string, error) {
	logger := log.FromContext(ctx)

	fileList, err := fe.podExecutor.exec(k8sExecPod.ObjectKey(), "/bin/bash", "-c", "/bin/ls /k8s/ || true")
	if err != nil {
		return nil, err
	}

	resultDocs := make(map[string]string)
	if fileList == "" || strings.Contains(fileList, "No such file or directory") || strings.Contains(fileList, "total 0") {
		logger.Info("No custom K8s resource files found")
		return resultDocs, nil
	}

	for _, file := range strings.Split(fileList, " ") {
		trimmedFile := doguCustomK8sResourceDirectory + strings.TrimSpace(file)
		logger.Info("Reading k8s resource " + trimmedFile)

		fileContent, err := fe.podExecutor.exec(k8sExecPod.ObjectKey(), "/bin/cat", trimmedFile)
		if err != nil {
			return nil, err
		}

		resultDocs[trimmedFile] = fileContent
	}

	return resultDocs, nil
}

type defaultPodExecutor struct {
	config    *rest.Config
	clientset kubernetes.Interface
}

func (pe *defaultPodExecutor) exec(execPodKey *client.ObjectKey, cmdArgs ...string) (stdOut string, err error) {
	execPod := newExecPod(pe.config, pe.clientset, execPodKey)

	out, _, err := execPod.execCmd(cmdArgs)
	if err != nil {
		return "", fmt.Errorf("could not enumerate K8s resources in ExecPod %s with command '%s': %w",
			execPodKey.Name, strings.Join(cmdArgs, " "), err)
	}

	return out.String(), nil
}

// podExec executes commands in a running K8s container
type podExec struct {
	clientset     kubernetes.Interface
	restConfig    *rest.Config
	namespace     string
	podName       string
	containerName string
	restExecutor  exec.RemoteExecutor
}

func newExecPod(config *rest.Config, clientSet kubernetes.Interface, podExecKey *client.ObjectKey) *podExec {
	config.APIPath = "/api"
	config.GroupVersion = &schema.GroupVersion{Version: "v1"}
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}

	return &podExec{
		restConfig:    config,
		clientset:     clientSet,
		namespace:     podExecKey.Namespace,
		podName:       podExecKey.Name,
		containerName: podExecKey.Name,
		restExecutor:  &exec.DefaultRemoteExecutor{},
	}
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
		Executor:      p.restExecutor,
		PodClient:     p.clientset.CoreV1(),
		GetPodTimeout: 0,
		Config:        p.restConfig,
	}

	err = options.Run()
	if err != nil {
		return nil, nil, fmt.Errorf("could not run exec operation: %w", err)
	}

	return out, errOut, nil
}
