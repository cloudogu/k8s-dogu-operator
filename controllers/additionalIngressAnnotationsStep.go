package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/annotation"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const requeueAfterAdditionalIngressAnnotations = 5 * time.Second

type AdditionalIngressAnnotationsStep struct {
	client                                  client.Client
	doguAdditionalIngressAnnotationsManager doguAdditionalIngressAnnotationsManager
}

func NewAdditionalIngressAnnotationsStep(mgrSet *util.ManagerSet, client client.Client, additionalIngressManager doguAdditionalIngressAnnotationsManager) *AdditionalIngressAnnotationsStep {
	return &AdditionalIngressAnnotationsStep{
		client:                                  client,
		doguAdditionalIngressAnnotationsManager: additionalIngressManager,
	}
}

func (aias *AdditionalIngressAnnotationsStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	ingressAnnotationsChanged, err := aias.checkForAdditionalIngressAnnotations(ctx, doguResource)
	if err != nil {
		return 0, err
	}
	if ingressAnnotationsChanged {
		err = aias.doguAdditionalIngressAnnotationsManager.SetDoguAdditionalIngressAnnotations(ctx, doguResource)
	}
	return 0, err
}

func (aias *AdditionalIngressAnnotationsStep) checkForAdditionalIngressAnnotations(ctx context.Context, doguResource *v2.Dogu) (bool, error) {
	doguService := &v1.Service{}
	err := aias.client.Get(ctx, doguResource.GetObjectKey(), doguService)
	if err != nil {
		return false, fmt.Errorf("failed to get service of dogu [%s]: %w", doguResource.Name, err)
	}

	annotationsJson, exists := doguService.Annotations[annotation.AdditionalIngressAnnotationsAnnotation]
	annotations := v2.IngressAnnotations(nil)
	if exists {
		err = json.Unmarshal([]byte(annotationsJson), &annotations)
		if err != nil {
			return false, fmt.Errorf("failed to get additional ingress annotations from service of dogu [%s]: %w", doguResource.Name, err)
		}
	}

	if reflect.DeepEqual(annotations, doguResource.Spec.AdditionalIngressAnnotations) {
		return false, nil
	} else {
		return true, nil
	}
}
