package controllers

import (
	"context"
	"fmt"
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

	lsCommand := resource.NewShellCommand("/bin/sh", "-c", "/bin/ls /k8s/ || true")
	fileList, err := k8sExecPod.Exec(ctx, lsCommand)

	logger.Info(fmt.Sprintf("ExecPod file list results in '%s'", fileList))

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

		catCommand := resource.NewShellCommand("/bin/cat", trimmedFile)
		fileContent, err := k8sExecPod.Exec(ctx, catCommand)
		if err != nil {
			return nil, err
		}

		resultDocs[trimmedFile] = fileContent
	}

	return resultDocs, nil
}
