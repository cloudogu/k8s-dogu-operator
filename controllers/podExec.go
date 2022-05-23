package controllers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/exec"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

// podExec creates a container pod for a given
type podExec struct {
	RestConfig *rest.Config
	*kubernetes.Clientset
	Namespace     string
	PodName       string
	ContainerName string
}

func newPodExec(ctx context.Context, namespace, podname, containername string) (*podExec, error) {
	logger := log.FromContext(ctx)
	logger.Info("Creating new podExec")
	config := ctrl.GetConfigOrDie()

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not create podExec")
	}
	config.APIPath = "/api"
	config.GroupVersion = &schema.GroupVersion{Version: "v1"}
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}
	return &podExec{
		RestConfig:    config,
		Clientset:     clientSet,
		Namespace:     namespace,
		PodName:       podname,
		ContainerName: containername,
	}, nil
}

// execCmd executes arbitrary commands in a pod container.
func (p *podExec) execCmd(command []string) (*bytes.Buffer, *bytes.Buffer, *bytes.Buffer, error) {
	ioStreams, in, out, errOut := genericclioptions.NewTestIOStreams()
	options := &exec.ExecOptions{
		StreamOptions: exec.StreamOptions{
			Namespace:       p.Namespace,
			PodName:         p.PodName,
			ContainerName:   p.ContainerName,
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
		Config:        p.RestConfig,
	}

	err := options.Run()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not run exec operation: %w", err)
	}

	return in, out, errOut, nil
}

// PodFile
// Implement Read and Write interface
type PodFile struct {
	Path string
	*podExec
}

func NewPodFile(path string, podexec *podExec) *PodFile {
	return &PodFile{
		Path:    path,
		podExec: podexec,
	}
}

// Read from Path to b []byte
func (pf *PodFile) Read(b []byte) (n int, err error) {
	buf := bytes.NewBuffer([]byte{})
	written, err := pf.downloadFile(buf)
	if err != nil {
		return 0, err
	}
	copy(b, buf.Bytes())
	return int(written), io.EOF
}

func (pf *PodFile) downloadFile(w io.Writer) (int64, error) {
	logger := log.FromContext(context.Background())
	logger.Info(fmt.Sprintf("looking in namespace %s", pf.Namespace))
	logger.Info(fmt.Sprintf("looking for pod %s", pf.PodName))
	logger.Info(fmt.Sprintf("looking for container %s", pf.ContainerName))
	logger.Info(fmt.Sprintf("looking for file %s", pf.Path))

	options := &exec.ExecOptions{}
	errOut := bytes.NewBuffer([]byte{})
	reader, writer := io.Pipe()

	options.StreamOptions = exec.StreamOptions{
		Namespace:     pf.Namespace,
		PodName:       pf.PodName,
		ContainerName: pf.ContainerName,
		IOStreams: genericclioptions.IOStreams{
			In:     nil,
			Out:    writer,
			ErrOut: errOut,
		},
	}
	options.Executor = &exec.DefaultRemoteExecutor{}
	options.Namespace = pf.Namespace
	options.PodName = pf.PodName
	options.ContainerName = pf.ContainerName
	options.Config = pf.podExec.RestConfig
	options.PodClient = pf.podExec.Clientset.CoreV1()
	options.Command = []string{"/bin/cat", pf.Path}

	go func(options *exec.ExecOptions, writer *io.PipeWriter) {
		defer writer.Close()
		err := options.Run()
		if err != nil {
			logger.Error(err, fmt.Sprintf("oh noez! something went wrong during '%s': %w", strings.Join(options.Command, " "), err))
		}
	}(options, writer)

	return io.Copy(w, reader)
}
