package upgrade

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/flux"
	fluxhelm "github.com/fluxcd/helm-controller/api/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Kind describes how the requested version of a dogu relates to its installed version.
type Kind int

const (
	// NoChange means the dogu is not installed yet or the requested version is installed.
	NoChange Kind = iota
	// Upgrade means the requested version is newer than the installed version.
	Upgrade
	// Downgrade means the requested version is older than the installed version.
	Downgrade
	// InvalidVersion means the requested or the installed version changed but is not a valid semantic version.
	InvalidVersion
)

// Change describes the version change of a dogu.
type Change struct {
	Kind Kind
	// Installed is the deployed chart version, empty if the dogu was never deployed.
	Installed string
	// Target is the version requested by the dogu resource.
	Target string
}

// Detector derives the version Change of a dogu from the deployed revisions of its HelmRelease.
type Detector struct {
	client client.Reader
}

// NewDetector creates a Detector.
func NewDetector(c client.Reader) *Detector {
	return &Detector{client: c}
}

// Detect returns the version Change of the given dogu.
func (d *Detector) Detect(ctx context.Context, dogu *v3beta1.Dogu) (Change, error) {
	change := Change{Kind: NoChange, Target: dogu.Spec.Version}

	release := &fluxhelm.HelmRelease{}
	key := client.ObjectKey{Namespace: dogu.Namespace, Name: dogu.Spec.Name}
	err := d.client.Get(ctx, key, release)
	if apierrors.IsNotFound(err) {
		return change, nil
	}

	if err != nil {
		return change, fmt.Errorf("failed to get HelmRelease %s: %w", key, err)
	}

	installed, deployed := flux.DeployedChartVersion(release)
	if !deployed {
		return change, nil
	}

	change.Installed = installed
	if installed == change.Target {
		return change, nil
	}

	installedVersion, installedErr := semver.NewVersion(installed)
	targetVersion, targetErr := semver.NewVersion(change.Target)
	if installedErr != nil || targetErr != nil {
		change.Kind = InvalidVersion
		return change, nil
	}

	switch targetVersion.Compare(installedVersion) {
	case 1:
		change.Kind = Upgrade
	case -1:
		change.Kind = Downgrade
	}

	return change, nil
}
