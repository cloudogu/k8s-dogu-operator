package controllers

import (
	"context"
	"strings"

	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const doguCustomK8sResourceDirectory = "/k8s/"

type podFileExtractor struct {
	k8sClient client.Client
	config    *rest.Config
	clientSet kubernetes.Interface
}

func newPodFileExtractor(k8sClient client.Client, restConfig *rest.Config, clientSet kubernetes.Interface) *podFileExtractor {
	return &podFileExtractor{
		k8sClient: k8sClient,
		config:    restConfig,
		clientSet: clientSet,
	}
}

// ExtractK8sResourcesFromContainer enumerates K8s resources and returns them in a map filename->content. The map will be
// empty if there are no files.
func (fe *podFileExtractor) ExtractK8sResourcesFromContainer(ctx context.Context, k8sExecPod util.ExecPod) (map[string]string, error) {
	logger := log.FromContext(ctx)

	lsCommand := resource.ShellCommand{
		Command: "/bin/bash",
		Args:    []string{"-c", "/bin/ls /k8s/ || true"},
	}
	fileList, _, err := k8sExecPod.Exec(&lsCommand)
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

		catCommand := resource.ShellCommand{
			Command: "/bin/cat",
			Args:    []string{trimmedFile},
		}
		fileContent, _, err := k8sExecPod.Exec(&catCommand)
		if err != nil {
			return nil, err
		}

		resultDocs[trimmedFile] = fileContent
	}

	return resultDocs, nil
}
