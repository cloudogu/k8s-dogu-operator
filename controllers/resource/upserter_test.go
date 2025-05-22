package resource

import (
	"context"
	"fmt"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/mock"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewUpserter(t *testing.T) {
	// given
	mockClient := newMockK8sClient(t)
	mockResourceGenerator := NewMockDoguResourceGenerator(t)

	// when
	upserter := NewUpserter(mockClient, mockResourceGenerator, true)

	// then
	require.NotNil(t, upserter)
	assert.Equal(t, mockClient, upserter.client)
	assert.Equal(t, upserter.networkPoliciesEnabled, true)
	require.NotNil(t, upserter.generator)
}

func Test_upserter_updateOrInsert(t *testing.T) {
	t.Run("fail when using different types of objects", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		upserter := upserter{}

		// when
		err := upserter.updateOrInsert(context.Background(), doguResource.GetObjectKey(), nil, &appsv1.StatefulSet{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "upsert type must be a valid pointer to an K8s resource")
	})
	t.Run("should fail on incompatible input types", func(t *testing.T) {
		// given
		depl := &appsv1.Deployment{}
		svc := &v1.Service{}
		doguResource := readLdapDoguResource(t)
		sut := upserter{}

		// when
		err := sut.updateOrInsert(context.Background(), doguResource.GetObjectKey(), depl, svc)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "incompatible types provided (*Deployment != *Service)")
	})

	t.Run("should update existing pcv when no controller reference is set", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		existingDeployment := readLdapDoguExpectedDeployment(t)
		// the test should override the replication count back to 1
		existingDeployment.Spec.Replicas = pointer.Int32(10)
		testClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource, existingDeployment).Build()
		upserter := upserter{client: testClient}

		// when
		upsertedDeployment := readLdapDoguExpectedDeployment(t)
		err := upserter.updateOrInsert(context.Background(), doguResource.GetObjectKey(), &appsv1.Deployment{}, upsertedDeployment)

		// then
		require.NoError(t, err)

		afterUpsert := &appsv1.Deployment{}
		err = testClient.Get(context.Background(), doguResource.GetObjectKey(), afterUpsert)
		assert.Nil(t, afterUpsert.Spec.Replicas)
		// mock assert happens during cleanup
	})
}

func Test_upserter_UpsertDoguDeployment(t *testing.T) {
	ctx := context.Background()
	t.Run("fail on error when generating resource", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		mockClient := newMockK8sClient(t)
		generator := NewMockDoguResourceGenerator(t)
		generator.EXPECT().CreateDoguDeployment(ctx, doguResource, dogu).Return(nil, assert.AnError)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguDeployment(ctx, doguResource, dogu, nil)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("fail when upserting", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		mockClient := newMockK8sClient(t)
		mockClient.EXPECT().Get(ctx, doguResource.GetObjectKey(), &appsv1.Deployment{}).Return(assert.AnError)

		generator := NewMockDoguResourceGenerator(t)
		generator.EXPECT().CreateDoguDeployment(ctx, doguResource, dogu).Return(readLdapDoguExpectedDeployment(t), nil)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguDeployment(ctx, doguResource, dogu, nil)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("successfully upsert deployment", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		testClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource).Build()

		generator := NewMockDoguResourceGenerator(t)
		generatedDeployment := readLdapDoguExpectedDeployment(t)
		generator.EXPECT().CreateDoguDeployment(ctx, doguResource, dogu).Return(generatedDeployment, nil)
		upserter := upserter{
			client:    testClient,
			generator: generator,
		}
		deploymentPatch := func(deployment *appsv1.Deployment) {
			deployment.Labels["test"] = "testvalue"
		}

		// when
		doguDeployment, err := upserter.UpsertDoguDeployment(ctx, doguResource, dogu, deploymentPatch)

		// then
		require.NoError(t, err)
		expectedDeployment := readLdapDoguExpectedDeployment(t)
		expectedDeployment.ResourceVersion = "1"
		expectedDeployment.Labels["test"] = "testvalue"
		assert.Equal(t, expectedDeployment, doguDeployment)
	})
}

func Test_upserter_UpsertDoguPVCs(t *testing.T) {
	t.Run("fail when pvc already exists and retrier timeouts", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		mockClient := newMockK8sClient(t)
		key := doguResource.GetObjectKey()
		mockClient.EXPECT().Get(context.Background(), key, &v1.PersistentVolumeClaim{}).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
			pvc := object.(*v1.PersistentVolumeClaim)
			now := metav1.Now()
			pvc.SetDeletionTimestamp(&now)

			return nil
		})

		generator := NewMockDoguResourceGenerator(t)
		generator.EXPECT().CreateDoguPVC(doguResource).Return(readLdapDoguExpectedDoguPVC(t), nil)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}
		oldTries := maximumTriesWaitForExistingPVC
		maximumTriesWaitForExistingPVC = 2
		defer func() {
			maximumTriesWaitForExistingPVC = oldTries
		}()

		// when
		_, err := upserter.UpsertDoguPVCs(context.Background(), doguResource, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to wait for existing pvc ldap to terminate: the maximum number of retries was reached: pvc ldap still exists")
	})

	t.Run("should throw an error if the resource generator fails to generate a dogu pvc", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		mockClient := newMockK8sClient(t)

		generator := NewMockDoguResourceGenerator(t)
		generator.On("CreateDoguPVC", doguResource).Return(nil, assert.AnError)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguPVCs(context.Background(), doguResource, dogu)

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to generate pvc")
	})

	t.Run("fail when upserting a dogu pvc", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		mockClient := newMockK8sClient(t)

		mockClient.On("Get", context.Background(), doguResource.GetObjectKey(), &v1.PersistentVolumeClaim{}).Return(assert.AnError)

		generator := NewMockDoguResourceGenerator(t)
		generator.On("CreateDoguPVC", doguResource).Return(readLdapDoguExpectedDoguPVC(t), nil)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguPVCs(context.Background(), doguResource, dogu)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("success when upserting a new dogu pvc", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)

		testClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource).Build()

		generator := NewMockDoguResourceGenerator(t)
		expectedDoguPVC := readLdapDoguExpectedDoguPVC(t)
		generator.On("CreateDoguPVC", doguResource).Return(expectedDoguPVC, nil)
		upserter := upserter{
			client:    testClient,
			generator: generator,
		}

		// when
		actualDoguPVC, err := upserter.UpsertDoguPVCs(context.Background(), doguResource, dogu)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedDoguPVC, actualDoguPVC)
	})

	t.Run("success when upserting a new dogu pvc when an old pvc is terminating", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		dogu := readLdapDogu(t)
		var doguPvc *v1.PersistentVolumeClaim
		now := metav1.Now()
		doguPvc = readLdapDoguExpectedDoguPVC(t)
		doguPvc.DeletionTimestamp = &now
		doguPvc.Finalizers = []string{"myFinalizer"}
		testClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource, doguPvc).WithStatusSubresource(&k8sv2.Dogu{}).Build()
		timer := time.NewTimer(time.Second * 5)
		go func() {
			<-timer.C
			patch := client.RawPatch(types.JSONPatchType, []byte(`[{"op": "remove", "path": "/metadata/finalizers"}]`))
			err := testClient.Patch(context.Background(), doguPvc, patch)
			require.NoError(t, err)
		}()

		generator := NewMockDoguResourceGenerator(t)
		expectedDoguPVC := readLdapDoguExpectedDoguPVC(t)
		generator.On("CreateDoguPVC", doguResource).Return(expectedDoguPVC, nil)
		upserter := upserter{
			client:    testClient,
			generator: generator,
		}

		// when
		actualDoguPVC, err := upserter.UpsertDoguPVCs(context.Background(), doguResource, dogu)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedDoguPVC, actualDoguPVC)
	})
}

func Test_upserter_UpsertDoguService(t *testing.T) {
	ctx := context.Background()
	t.Run("fail on error when generating resource", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		imageConfig := readLdapDoguImageConfig(t)

		mockClient := newMockK8sClient(t)
		generator := NewMockDoguResourceGenerator(t)
		generator.On("CreateDoguService", doguResource, ldapDogu, imageConfig).Return(nil, assert.AnError)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguService(ctx, doguResource, ldapDogu, imageConfig)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("fail when upserting", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		imageConfig := readLdapDoguImageConfig(t)

		mockClient := newMockK8sClient(t)
		mockClient.On("Get", ctx, doguResource.GetObjectKey(), &v1.Service{}).Return(assert.AnError)

		generator := NewMockDoguResourceGenerator(t)
		expectedService := readLdapDoguExpectedService(t)
		generator.On("CreateDoguService", doguResource, ldapDogu, imageConfig).Return(expectedService, nil)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguService(ctx, doguResource, ldapDogu, imageConfig)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("successfully upsert service", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		imageConfig := readLdapDoguImageConfig(t)

		testClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource).Build()

		generator := NewMockDoguResourceGenerator(t)
		expectedService := readLdapDoguExpectedService(t)
		generator.On("CreateDoguService", doguResource, ldapDogu, imageConfig).Return(expectedService, nil)
		upserter := upserter{
			client:    testClient,
			generator: generator,
		}

		// when
		actualService, err := upserter.UpsertDoguService(ctx, doguResource, ldapDogu, imageConfig)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedService, actualService)
	})

	t.Run("fail when upserting a service", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		ldapDogu := readLdapDogu(t)
		imageConfig := readLdapDoguImageConfig(t)

		mockClient := newMockK8sClient(t)
		mockClient.On("Get", context.Background(), doguResource.GetObjectKey(), &v1.Service{}).Return(assert.AnError)

		generator := NewMockDoguResourceGenerator(t)
		generator.On("CreateDoguService", doguResource, ldapDogu, imageConfig).Return(readLdapDoguExpectedService(t), nil)
		upserter := upserter{
			client:    mockClient,
			generator: generator,
		}

		// when
		_, err := upserter.UpsertDoguService(context.Background(), doguResource, ldapDogu, imageConfig)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
}

func Test_upserter_UpsertDoguNetworkPolicies(t *testing.T) {
	tests := []struct {
		name                              string
		doguName                          string
		doguDependencies                  []string
		componentDependencies             []string
		expectedNetworkPolicies           []string
		additionalExistingNetworkPolicies []string
		errorOnUpdate                     bool
		expectError                       bool
		networkPoliciesEnabled            bool
	}{
		{
			name:                    "creates deny all policy for dogu without dependencies",
			doguName:                "postgresql",
			doguDependencies:        []string{},
			expectedNetworkPolicies: []string{"postgresql-deny-all"},
			networkPoliciesEnabled:  true,
		},
		{
			name:     "creates all dependencies for a dogu with ui (nginx included)",
			doguName: "redmine",
			doguDependencies: []string{
				"postgresql",
				"nginx-ingress",
				"nginx-static",
				"cas",
				"postfix",
			},
			componentDependencies: []string{
				"k8s-ces-control",
			},
			expectedNetworkPolicies: []string{
				"redmine-deny-all",
				"redmine-ingress",
				"redmine-dependency-dogu-cas",
				"redmine-dependency-dogu-postfix",
				"redmine-dependency-dogu-postgresql",
				"redmine-dependency-component-k8s-ces-control",
			},
			networkPoliciesEnabled: true,
		},
		{
			name:     "creates all dependencies for a dogu without ui",
			doguName: "redmine",
			doguDependencies: []string{
				"postgresql",
				"cas",
				"postfix",
			},
			expectedNetworkPolicies: []string{
				"redmine-deny-all",
				"redmine-dependency-dogu-cas",
				"redmine-dependency-dogu-postfix",
				"redmine-dependency-dogu-postgresql",
			},
			networkPoliciesEnabled: true,
		},
		{
			name:     "fails on error with update",
			doguName: "redmine",
			doguDependencies: []string{
				"postgresql",
				"nginx-ingress",
				"nginx-static",
				"cas",
				"postfix",
			},
			expectedNetworkPolicies: []string{
				"redmine-deny-all",
				"redmine-ingress",
				"redmine-dependency-dogu-cas",
				"redmine-dependency-dogu-postfix",
				"redmine-dependency-dogu-postgresql",
			},
			errorOnUpdate:          true,
			expectError:            true,
			networkPoliciesEnabled: true,
		},
		{
			name:     "deletes superfluous network policies",
			doguName: "redmine",
			doguDependencies: []string{
				"postgresql",
				"nginx-ingress",
				"nginx-static",
				"cas",
				"postfix",
			},
			expectedNetworkPolicies: []string{
				"redmine-deny-all",
				"redmine-ingress",
				"redmine-dependency-dogu-cas",
				"redmine-dependency-dogu-postfix",
				"redmine-dependency-dogu-postgresql",
			},
			additionalExistingNetworkPolicies: []string{
				"redmine-dependency-dogu-scm",
				"redmine-dependency-dogu-admin",
			},
			networkPoliciesEnabled: true,
		},
		{
			name:     "no network policies created when networkPoliciesEnabled=false",
			doguName: "redmine",
			doguDependencies: []string{
				"postgresql",
				"nginx-ingress",
				"nginx-static",
				"cas",
				"postfix",
			},
			expectedNetworkPolicies: []string{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var dependencies []cesappcore.Dependency
			for _, dep := range test.doguDependencies {
				dependencies = append(dependencies, cesappcore.Dependency{
					Type: dependencyTypeDogu,
					Name: dep,
				})
			}
			for _, dep := range test.componentDependencies {
				dependencies = append(dependencies, cesappcore.Dependency{
					Type: dependencyTypeComponent,
					Name: dep,
				})
			}
			dogu := &cesappcore.Dogu{
				Name:         test.doguName,
				Dependencies: dependencies,
			}
			doguResource := &k8sv2.Dogu{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       k8sv2.DoguSpec{},
				Status:     k8sv2.DoguStatus{},
			}

			times := len(test.expectedNetworkPolicies)
			mockClient := newMockK8sClient(t)
			if times > 0 {
				mockClient.EXPECT().Get(context.Background(), mock.Anything, mock.AnythingOfType("*v1.NetworkPolicy")).Return(nil).Times(times)
			}

			if test.networkPoliciesEnabled {
				mockClient.EXPECT().List(context.Background(), mock.Anything, client.MatchingLabels{"dogu.name": dogu.GetSimpleName()}).Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
					newList := list.(*netv1.NetworkPolicyList)
					allExpectedPolicies := append(test.expectedNetworkPolicies, test.additionalExistingNetworkPolicies...)
					newList.Items = make([]netv1.NetworkPolicy, len(allExpectedPolicies))

					for _, policy := range allExpectedPolicies {
						if strings.Contains(policy, "deny-all") || strings.Contains(policy, "ingress") {
							continue
						}
						doguDependencyPrefix := fmt.Sprintf("%s-dependency-dogu-", dogu.GetSimpleName())
						componentDependencyPrefix := fmt.Sprintf("%s-dependency-component-", dogu.GetSimpleName())
						newList.Items = append(newList.Items, netv1.NetworkPolicy{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									depenendcyLabel: strings.TrimPrefix(strings.TrimPrefix(policy, doguDependencyPrefix), componentDependencyPrefix),
								},
							},
						})
					}
				}).Return(nil).Once()
			}

			if len(test.additionalExistingNetworkPolicies) > 0 {
				mockClient.EXPECT().Delete(context.Background(), mock.Anything, mock.Anything).Times(len(test.additionalExistingNetworkPolicies)).Return(nil)
			}

			var errResult error
			if test.errorOnUpdate {
				errResult = assert.AnError
			}

			var actualCalledPolicies []string
			for range test.expectedNetworkPolicies {
				mockClient.EXPECT().Update(context.Background(), mock.AnythingOfType("*v1.NetworkPolicy")).Run(func(ctx context.Context, clientObject client.Object, opts ...client.UpdateOption) {
					policy, ok := clientObject.(*netv1.NetworkPolicy)
					if !ok {
						t.Error("the arg 1 passed to Update was not of type *v1.NetworkPolicy")
					}
					if !slices.Contains(test.expectedNetworkPolicies, policy.Name) {
						t.Errorf("the network policy %s was created but not expected", policy)
					}
					actualCalledPolicies = append(actualCalledPolicies, policy.Name)
				}).Return(errResult).Once()
			}

			generator := NewMockDoguResourceGenerator(t)

			ups := upserter{
				client:                 mockClient,
				generator:              generator,
				networkPoliciesEnabled: test.networkPoliciesEnabled,
			}

			err := ups.UpsertDoguNetworkPolicies(context.Background(), doguResource, dogu)
			if !test.expectError {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			for _, expectedPolicy := range test.expectedNetworkPolicies {
				if !slices.Contains(actualCalledPolicies, expectedPolicy) {
					assert.Fail(t, fmt.Sprintf("the policy '%s' was expected but not created", expectedPolicy))
				}
			}

			mockClient.AssertExpectations(t)

		})
	}
}
