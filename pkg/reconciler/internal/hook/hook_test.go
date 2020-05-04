package hook_test

import (
	"strings"

	"github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/joelanford/helm-operator/pkg/hook"
	"github.com/joelanford/helm-operator/pkg/internal/sdk/fake"
	sdkhandler "github.com/joelanford/helm-operator/pkg/internal/sdk/handler"
	internalhook "github.com/joelanford/helm-operator/pkg/reconciler/internal/hook"
)

var _ = Describe("Hook", func() {
	Describe("dependentResourceWatcher", func() {
		var (
			drw   hook.PostHook
			c     *fake.Controller
			rm    *meta.DefaultRESTMapper
			owner *unstructured.Unstructured
			rel   *release.Release
			log   *testing.TestLogger
		)

		BeforeEach(func() {
			rm = meta.NewDefaultRESTMapper([]schema.GroupVersion{})
			c = &fake.Controller{}
			log = &testing.TestLogger{}
		})

		Context("with unknown APIs", func() {
			BeforeEach(func() {
				owner = &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"metadata": map[string]interface{}{
							"name":      "testDeployment",
							"namespace": "ownerNamespace",
						},
					},
				}
				rel = &release.Release{
					Manifest: strings.Join([]string{rsOwnerNamespace}, "---\n"),
				}
				drw = internalhook.NewDependentResourceWatcher(c, rm, owner)
			})
			It("should fail with unknown owner kind", func() {
				Expect(drw.Exec(nil, rel, log)).To(MatchError(&meta.NoKindMatchError{
					GroupKind:        schema.GroupKind{Group: "apps", Kind: "Deployment"},
					SearchedVersions: []string{"v1"},
				}))
			})
			It("should fail with unknown dependent kind", func() {
				rm.Add(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}, meta.RESTScopeNamespace)
				Expect(drw.Exec(nil, rel, log)).To(MatchError(&meta.NoKindMatchError{
					GroupKind:        schema.GroupKind{Group: "apps", Kind: "ReplicaSet"},
					SearchedVersions: []string{"v1"},
				}))
			})
		})

		Context("with known APIs", func() {
			BeforeEach(func() {
				rm.Add(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}, meta.RESTScopeNamespace)
				rm.Add(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"}, meta.RESTScopeNamespace)
				rm.Add(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"}, meta.RESTScopeNamespace)
				rm.Add(schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}, meta.RESTScopeRoot)
				rm.Add(schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}, meta.RESTScopeRoot)
			})

			It("should watch resource kinds only once each", func() {
				owner = &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "rbac.authorization.k8s.io/v1",
						"kind":       "ClusterRole",
						"metadata": map[string]interface{}{
							"name": "testClusterRole",
						},
					},
				}
				rel = &release.Release{
					Manifest: strings.Join([]string{clusterRole, clusterRole, rsOwnerNamespace, rsOwnerNamespace}, "---\n"),
				}
				drw = internalhook.NewDependentResourceWatcher(c, rm, owner)
				Expect(drw.Exec(nil, rel, log)).To(Succeed())
				Expect(c.WatchCalls).To(HaveLen(2))
				Expect(c.WatchCalls[0].Handler).To(BeAssignableToTypeOf(&handler.EnqueueRequestForOwner{}))
				Expect(c.WatchCalls[1].Handler).To(BeAssignableToTypeOf(&handler.EnqueueRequestForOwner{}))
			})

			Context("when the owner is cluster-scoped", func() {
				BeforeEach(func() {
					owner = &unstructured.Unstructured{
						Object: map[string]interface{}{
							"apiVersion": "rbac.authorization.k8s.io/v1",
							"kind":       "ClusterRole",
							"metadata": map[string]interface{}{
								"name": "testClusterRole",
							},
						},
					}
				})
				It("should watch namespace-scoped resources with ownerRef handler", func() {
					rel = &release.Release{
						Manifest: strings.Join([]string{rsOwnerNamespace, ssOtherNamespace}, "---\n"),
					}
					drw = internalhook.NewDependentResourceWatcher(c, rm, owner)
					Expect(drw.Exec(nil, rel, log)).To(Succeed())
					Expect(c.WatchCalls).To(HaveLen(2))
					Expect(c.WatchCalls[0].Handler).To(BeAssignableToTypeOf(&handler.EnqueueRequestForOwner{}))
					Expect(c.WatchCalls[1].Handler).To(BeAssignableToTypeOf(&handler.EnqueueRequestForOwner{}))

				})
				It("should watch cluster-scoped resources with ownerRef handler", func() {
					rel = &release.Release{
						Manifest: strings.Join([]string{clusterRole, clusterRoleBinding}, "---\n"),
					}
					drw = internalhook.NewDependentResourceWatcher(c, rm, owner)
					Expect(drw.Exec(nil, rel, log)).To(Succeed())
					Expect(c.WatchCalls).To(HaveLen(2))
					Expect(c.WatchCalls[0].Handler).To(BeAssignableToTypeOf(&handler.EnqueueRequestForOwner{}))
					Expect(c.WatchCalls[1].Handler).To(BeAssignableToTypeOf(&handler.EnqueueRequestForOwner{}))
				})
			})

			Context("when the owner is namespace-scoped", func() {
				BeforeEach(func() {
					owner = &unstructured.Unstructured{
						Object: map[string]interface{}{
							"apiVersion": "apps/v1",
							"kind":       "Deployment",
							"metadata": map[string]interface{}{
								"name":      "testDeployment",
								"namespace": "ownerNamespace",
							},
						},
					}
				})

				It("should watch namespace-scoped dependent resources in the same namespace with ownerRef handler", func() {
					rel = &release.Release{
						Manifest: strings.Join([]string{rsOwnerNamespace}, "---\n"),
					}
					drw = internalhook.NewDependentResourceWatcher(c, rm, owner)
					Expect(drw.Exec(nil, rel, log)).To(Succeed())
					Expect(c.WatchCalls).To(HaveLen(1))
					Expect(c.WatchCalls[0].Handler).To(BeAssignableToTypeOf(&handler.EnqueueRequestForOwner{}))
				})

				It("should watch cluster-scoped resources with annotation handler", func() {
					rel = &release.Release{
						Manifest: strings.Join([]string{clusterRole}, "---\n"),
					}
					drw = internalhook.NewDependentResourceWatcher(c, rm, owner)
					Expect(drw.Exec(nil, rel, log)).To(Succeed())
					Expect(c.WatchCalls).To(HaveLen(1))
					Expect(c.WatchCalls[0].Handler).To(BeAssignableToTypeOf(&sdkhandler.EnqueueRequestForAnnotation{}))
				})

				It("should watch namespace-scoped resources in a different namespace with annotation handler", func() {
					rel = &release.Release{
						Manifest: strings.Join([]string{ssOtherNamespace}, "---\n"),
					}
					drw = internalhook.NewDependentResourceWatcher(c, rm, owner)
					Expect(drw.Exec(nil, rel, log)).To(Succeed())
					Expect(c.WatchCalls).To(HaveLen(1))
					Expect(c.WatchCalls[0].Handler).To(BeAssignableToTypeOf(&sdkhandler.EnqueueRequestForAnnotation{}))
				})
			})
		})
	})
})

var (
	rsOwnerNamespace = `
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: testReplicaSet
  namespace: ownerNamespace
`
	ssOtherNamespace = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: otherTestStatefulSet
  namespace: otherNamespace
`
	clusterRole = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: testClusterRole
`
	clusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: testClusterRoleBinding
`
)