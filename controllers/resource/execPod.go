package resource

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/rand"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// maxTries controls the maximum number of waiting intervals between requesting an exec pod and its actual
// instantiation. The waiting time linearly increases each iteration.
var maxTries = 20

type suffixGenerator interface {
	// String returns a random suffix string with the given length
	String(length int) string
}

// ExecPod provides features to handle files from a dogu image.
type ExecPod struct {
	client       client.Client
	suffixGen    suffixGenerator
	k8sNamespace string
	doguResource *k8sv1.Dogu
	dogu         *core.Dogu
	podName      string
	deleteSpec   *corev1.Pod
}

func NewExecPod(k8sNamespace string, doguResource *k8sv1.Dogu, dogu *core.Dogu, client client.Client) *ExecPod {
	suffixGen := &defaultSufficeGenerator{}
	return &ExecPod{
		client:       client,
		suffixGen:    suffixGen,
		k8sNamespace: k8sNamespace,
		doguResource: doguResource,
		dogu:         dogu,
	}
}

// Create adds a new exec pod to the cluster.
func (ep *ExecPod) Create(ctx context.Context) error {
	logger := log.FromContext(ctx)

	ep.podName = generatePodName(ep.dogu, ep.suffixGen)

	execPodSpec, err := ep.createPod(ep.k8sNamespace, ep.podName)
	if err != nil {
		return err
	}

	logger.Info("Creating new exec pod " + ep.podName)
	err = ep.client.Create(ctx, execPodSpec)
	if err != nil {
		return err
	}

	err = ep.waitForPodToSpawn(ctx)
	if err != nil {
		return err
	}

	return nil
}

func generatePodName(dogu *core.Dogu, generator suffixGenerator) string {
	return fmt.Sprintf("%s-%s-%s", dogu.GetSimpleName(), "execpod", generator.String(6))
}

func (ep *ExecPod) createPod(k8sNamespace string, containerName string) (*corev1.Pod, error) {
	image := ep.dogu.Image + ":" + ep.dogu.Version
	// command is of no importance because the pod will be killed after success
	doNothingCommand := []string{"/bin/sleep", "60"}
	labels := map[string]string{"app": "ces", "dogu": containerName}

	pullPolicy := corev1.PullIfNotPresent
	if config.Stage == config.StageDevelopment {
		pullPolicy = corev1.PullAlways
	}

	podSpec := &corev1.Pod{
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
					ImagePullPolicy: pullPolicy,
				},
			},
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "k8s-dogu-operator-docker-registry"},
			},
		},
	}

	err := ctrl.SetControllerReference(ep.doguResource, podSpec, ep.client.Scheme())
	if err != nil {
		return nil, fmt.Errorf("failed to set controller reference to exec pod %s: %w", containerName, err)
	}
	return podSpec, nil
}

func (ep *ExecPod) waitForPodToSpawn(ctx context.Context) error {
	logger := log.FromContext(ctx)

	execPodKey := ep.ObjectKey()

	lePod := corev1.Pod{}
	containerPodName := execPodKey.Name

	for i := 1; i <= maxTries; i++ {
		if i >= maxTries {
			return fmt.Errorf("quitting dogu installation because exec pod %s could not be found", containerPodName)
		}

		err := ep.client.Get(ctx, *execPodKey, &lePod)
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

// Delete deletes the exec pod from the cluster.
func (ep *ExecPod) Delete(ctx context.Context) error {
	logger := log.FromContext(ctx)

	logger.Info("Cleaning up exec pod ", ep.podName)
	err := ep.client.Delete(ctx, ep.deleteSpec)
	if err != nil {
		err2 := fmt.Errorf("failed to delete custom dogu descriptor: %w", err)
		if !errors.IsNotFound(err) {
			return err2
		}

		logger.Error(err2, "Error deleting execPod ")
	}

	return nil
}

// PodName returns the name of the created exec pod resource.
func (ep *ExecPod) PodName() string {
	return ep.podName
}

func (ep *ExecPod) ObjectKey() *client.ObjectKey {
	return &client.ObjectKey{
		Namespace: ep.k8sNamespace,
		Name:      ep.podName,
	}
}

func sleep(logger logr.Logger, sleepIntervalInSec int) {
	logger.Info(fmt.Sprintf("Exec pod not found. Trying again in %d second(s)", sleepIntervalInSec))
	time.Sleep(time.Duration(sleepIntervalInSec) * time.Second) // linear rising backoff
}

type defaultSufficeGenerator struct{}

func (sg *defaultSufficeGenerator) String(suffixLength int) string {
	return rand.String(suffixLength)
}
