package exec

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/retry-lib/retry"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	quantity "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// deprecated
var maxWaitDuration = time.Minute * 10

const execPodLabel = "k8s.cloudogu.com/execPod"

// execPodFactory provides features to handle files from a dogu image.
type execPodFactory struct {
	client   client.Client
	executor CommandExecutor
}

// NewExecPodFactory creates a new ExecPod that enables command execution towards a pod.
func NewExecPodFactory(
	client client.Client,
	executor CommandExecutor,
) *execPodFactory {
	return &execPodFactory{
		client:   client,
		executor: executor,
	}
}

func execPodName(dogu *core.Dogu) string {
	return fmt.Sprintf("%s-%s", dogu.GetSimpleName(), "execpod")
}

// CreateBlocking adds a new exec pod to the cluster. It waits synchronously until the K8s pod resource exists.
// Deprecated, as we want our code to be non-blocking in the future.
func (ep *execPodFactory) CreateBlocking(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) error {
	execPodSpec, err := ep.createPod(doguResource, dogu)
	if err != nil {
		return err
	}

	err = ep.client.Create(ctx, execPodSpec)
	if err != nil {
		return err
	}

	err = ep.waitForPodToSpawn(ctx, doguResource, dogu)
	if err != nil {
		return err
	}

	return nil
}

// Create adds a new exec pod to the cluster.
func (ep *execPodFactory) Create(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) error {
	execPodSpec, err := ep.createPod(doguResource, dogu)
	if err != nil {
		return err
	}

	err = ep.client.Create(ctx, execPodSpec)
	if err != nil {
		return err
	}

	return nil
}

func (ep *execPodFactory) Exists(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) bool {
	_, err := ep.getPod(ctx, doguResource, dogu)
	return !errors.IsNotFound(err)
}

func (ep *execPodFactory) CheckReady(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) error {
	pod, err := ep.getPod(ctx, doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to get exec pod %q: %w", execPodName(dogu), err)
	}

	podStatus := pod.Status.Phase
	if podStatus == corev1.PodRunning {
		return nil
	}

	return fmt.Errorf("exec pod %q has status phase %q: not running", execPodName(dogu), podStatus)
}

func (ep *execPodFactory) createPod(doguResource *k8sv2.Dogu, dogu *core.Dogu) (*corev1.Pod, error) {
	image := dogu.Image + ":" + dogu.Version
	doNothingCommand := []string{"/bin/sleep", "infinity"}
	// set app name for completeness's sake so all generated resource can be selected (and possibly cleaned up) with our ces label.
	labels := resource.GetAppLabel()
	labels[execPodLabel] = dogu.GetSimpleName()

	pullPolicy := corev1.PullIfNotPresent
	if config.Stage == config.StageDevelopment {
		pullPolicy = corev1.PullAlways
	}

	automountServiceAccountToken := false

	podName := execPodName(dogu)
	podSpec := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:        podName,
			Namespace:   doguResource.Namespace,
			Labels:      labels,
			Annotations: make(map[string]string),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            podName,
					Image:           image,
					Command:         doNothingCommand,
					ImagePullPolicy: pullPolicy,
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceMemory: quantity.MustParse("105M"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceMemory: quantity.MustParse("105M"),
							corev1.ResourceCPU:    quantity.MustParse("15m"),
						},
					},
				},
			},
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "ces-container-registries"},
			},
			AutomountServiceAccountToken: &automountServiceAccountToken,
		},
	}

	err := ctrl.SetControllerReference(doguResource, podSpec, ep.client.Scheme())
	if err != nil {
		return nil, fmt.Errorf("failed to set controller reference to exec pod %q: %w", podName, err)
	}
	return podSpec, nil
}

// deprecated
func (ep *execPodFactory) waitForPodToSpawn(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	err := retry.OnErrorWithLimit(maxWaitDuration, retry.TestableRetryFunc, func() error {
		pod, err := ep.getPod(ctx, doguResource, dogu)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Error while finding exec pod %s. Trying again...", execPodName(dogu)))
			return &retry.TestableRetrierError{Err: err}
		}

		podStatus := pod.Status.Phase
		switch podStatus {
		case corev1.PodRunning:
			logger.Info("Found a ready exec pod " + pod.Name)
			return nil
		case corev1.PodFailed, corev1.PodSucceeded:
			return fmt.Errorf("quitting dogu installation because exec pod %s failed with status %s or did not come up in time", execPodName(dogu), podStatus)
		default:
			logger.Info(fmt.Sprintf("Found exec pod %s but with status phase %+v. Trying again...", execPodName(dogu), podStatus))
			return &retry.TestableRetrierError{Err: fmt.Errorf("found exec pod %s but with status phase %+v", pod.Name, podStatus)}
		}
	})
	if err != nil {
		return fmt.Errorf("failed to wait for exec pod %s to spawn: %w", execPodName(dogu), err)
	}

	return nil
}

func (ep *execPodFactory) getPod(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	err := ep.client.Get(ctx, types.NamespacedName{
		Namespace: doguResource.Namespace,
		Name:      execPodName(dogu),
	}, pod)

	return pod, err
}

// Delete deletes the exec pod from the cluster.
func (ep *execPodFactory) Delete(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) error {
	err := ep.client.DeleteAllOf(ctx, &corev1.Pod{}, client.MatchingLabels{execPodLabel: dogu.GetSimpleName()}, client.InNamespace(doguResource.Namespace))
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete execPodFactory %s: %w", execPodName(dogu), err)
		}
	}

	return nil
}

// Exec executes the given ShellCommand and returns any output to stdOut and stdErr.
func (ep *execPodFactory) Exec(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu, cmd ShellCommand) (*bytes.Buffer, error) {
	pod, err := ep.getPod(ctx, doguResource, dogu)
	if err != nil {
		return nil, fmt.Errorf("could not get pod: %w", err)
	}

	return ep.executor.ExecCommandForPod(ctx, pod, cmd, ContainersStarted)
}
