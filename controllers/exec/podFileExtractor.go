package exec

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const doguCustomK8sResourceDirectory = "/k8s/"

type podFileExtractor struct {
	factory ExecPodFactory
}

// NewPodFileExtractor creates a new pod file extractor that fetches files from a pod's container.
func NewPodFileExtractor(factory ExecPodFactory) FileExtractor {
	return &podFileExtractor{factory: factory}
}

// ExtractK8sResourcesFromExecPod enumerates K8s resources and returns them in a map filename->content. The map will be
// empty if there are no files.
func (fe *podFileExtractor) ExtractK8sResourcesFromExecPod(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) (map[string]string, error) {
	logger := log.FromContext(ctx)

	lsCommand := NewShellCommand("/bin/sh", "-c", "/bin/ls /k8s/ || true")
	fileListBuf, err := fe.factory.Exec(ctx, doguResource, dogu, lsCommand)
	if err != nil {
		return nil, err
	}

	fileList := fileListBuf.String()
	logger.Info(fmt.Sprintf("ExecPod file list results in '%s'", fileListBuf))

	resultDocs := make(map[string]string)
	if fileList == "" || strings.Contains(fileList, "No such file or directory") || strings.Contains(fileList, "total 0") {
		logger.Info("No custom K8s resource files found")
		return resultDocs, nil
	}

	for _, file := range strings.Split(fileList, " ") {
		trimmedFile := doguCustomK8sResourceDirectory + strings.TrimSpace(file)
		logger.Info("Reading k8s resource " + trimmedFile)

		catCommand := NewShellCommand("/bin/cat", trimmedFile)
		fileContent, err := fe.factory.Exec(ctx, doguResource, dogu, catCommand)
		if err != nil {
			return nil, err
		}

		resultDocs[trimmedFile] = fileContent.String()
	}

	return resultDocs, nil
}
