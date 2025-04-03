package main

import (
	"context"
	"testing"

	"dev.upbound.io/models/com/example/platform/v1alpha1"
	metav1 "dev.upbound.io/models/io/k8s/meta/v1"
	storagev1beta1 "dev.upbound.io/models/io/upbound/azure/storage/v1beta1"
	azv1beta1 "dev.upbound.io/models/io/upbound/azure/v1beta1"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"k8s.io/utils/ptr"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composite"
	"github.com/crossplane/function-sdk-go/response"
)

func TestRunFunction(t *testing.T) {
	type args struct {
		ctx context.Context
		req *fnv1.RunFunctionRequest
	}
	type want struct {
		rsp *fnv1.RunFunctionResponse
		err error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"ResourceGroupNotYetCreated": {
			reason: "If the resource group hasn't been created yet, only the resource group should be desired.",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Observed: &fnv1.State{
						Composite: toResource(&v1alpha1.XStorageBucket{
							Metadata: &metav1.ObjectMeta{
								Name: ptr.To("example-xr"),
							},
							Spec: &v1alpha1.XStorageBucketSpec{
								Parameters: &v1alpha1.XStorageBucketSpecParameters{
									Location:   ptr.To("us-east-1"),
									ACL:        ptr.To("private"),
									Versioning: ptr.To(false),
								},
							},
						}),
					},
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{{
						Severity: fnv1.Severity_SEVERITY_NORMAL,
						Message:  "waiting for resource group to be created",
						Target:   fnv1.Target_TARGET_COMPOSITE_AND_CLAIM.Enum(),
					}},
					Desired: &fnv1.State{
						Resources: map[string]*fnv1.Resource{
							"rg": toResource(&azv1beta1.ResourceGroup{
								APIVersion: ptr.To("azure.upbound.io/v1beta1"),
								Kind:       ptr.To("ResourceGroup"),
								Spec: &azv1beta1.ResourceGroupSpec{
									ForProvider: &azv1beta1.ResourceGroupSpecForProvider{
										Location: ptr.To("us-east-1"),
									},
								},
							}),
						},
					},
				},
			},
		},
		"ResourceGroupCreated": {
			reason: "If the resource group has been created, all resources should be desired.",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Observed: &fnv1.State{
						Composite: toResource(&v1alpha1.XStorageBucket{
							Metadata: &metav1.ObjectMeta{
								Name: ptr.To("example-xr"),
							},
							Spec: &v1alpha1.XStorageBucketSpec{
								Parameters: &v1alpha1.XStorageBucketSpecParameters{
									Location:   ptr.To("us-east-1"),
									ACL:        ptr.To("private"),
									Versioning: ptr.To(false),
								},
							},
						}),
						Resources: map[string]*fnv1.Resource{
							"rg": toResource(&azv1beta1.ResourceGroup{
								APIVersion: ptr.To("azure.upbound.io/v1beta1"),
								Kind:       ptr.To("ResourceGroup"),
								Metadata: &metav1.ObjectMeta{
									Annotations: &map[string]string{
										"crossplane.io/external-name": "super-group",
									},
								},
								Spec: &azv1beta1.ResourceGroupSpec{
									ForProvider: &azv1beta1.ResourceGroupSpecForProvider{
										Location: ptr.To("us-east-1"),
									},
								},
							}),
						},
					},
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta:    &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{},
					Desired: &fnv1.State{
						Resources: map[string]*fnv1.Resource{
							"rg": toResource(&azv1beta1.ResourceGroup{
								APIVersion: ptr.To("azure.upbound.io/v1beta1"),
								Kind:       ptr.To("ResourceGroup"),
								Spec: &azv1beta1.ResourceGroupSpec{
									ForProvider: &azv1beta1.ResourceGroupSpecForProvider{
										Location: ptr.To("us-east-1"),
									},
								},
							}),
							"acct": toResource(&storagev1beta1.Account{
								APIVersion: ptr.To("storage.azure.upbound.io/v1beta1"),
								Kind:       ptr.To("Account"),
								Metadata: &metav1.ObjectMeta{
									Annotations: &map[string]string{
										"crossplane.io/external-name": "examplexr",
									},
								},
								Spec: &storagev1beta1.AccountSpec{
									ForProvider: &storagev1beta1.AccountSpecForProvider{
										ResourceGroupName:               ptr.To("super-group"),
										AccountTier:                     ptr.To("Standard"),
										AccountReplicationType:          ptr.To("LRS"),
										Location:                        ptr.To("us-east-1"),
										InfrastructureEncryptionEnabled: ptr.To(true),
										BlobProperties: &[]storagev1beta1.AccountSpecForProviderBlobPropertiesItem{{
											VersioningEnabled: ptr.To(false),
										}},
									},
								},
							}),
							"cont": toResource(&storagev1beta1.Container{
								APIVersion: ptr.To("storage.azure.upbound.io/v1beta1"),
								Kind:       ptr.To("Container"),
								Spec: &storagev1beta1.ContainerSpec{
									ForProvider: &storagev1beta1.ContainerSpecForProvider{
										StorageAccountName:  ptr.To("examplexr"),
										ContainerAccessType: ptr.To("private"),
									},
								},
							}),
						},
					},
				},
			},
		},
		"ResourceGroupCreatedWithVersioning": {
			reason: "If the resource group has been created with versioning requested, all resources should be desired.",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Observed: &fnv1.State{
						Composite: toResource(&v1alpha1.XStorageBucket{
							Metadata: &metav1.ObjectMeta{
								Name: ptr.To("example-xr"),
							},
							Spec: &v1alpha1.XStorageBucketSpec{
								Parameters: &v1alpha1.XStorageBucketSpecParameters{
									Location:   ptr.To("us-east-1"),
									ACL:        ptr.To("private"),
									Versioning: ptr.To(true),
								},
							},
						}),
						Resources: map[string]*fnv1.Resource{
							"rg": toResource(&azv1beta1.ResourceGroup{
								APIVersion: ptr.To("azure.upbound.io/v1beta1"),
								Kind:       ptr.To("ResourceGroup"),
								Metadata: &metav1.ObjectMeta{
									Annotations: &map[string]string{
										"crossplane.io/external-name": "super-group",
									},
								},
								Spec: &azv1beta1.ResourceGroupSpec{
									ForProvider: &azv1beta1.ResourceGroupSpecForProvider{
										Location: ptr.To("us-east-1"),
									},
								},
							}),
						},
					},
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta:    &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{},
					Desired: &fnv1.State{
						Resources: map[string]*fnv1.Resource{
							"rg": toResource(&azv1beta1.ResourceGroup{
								APIVersion: ptr.To("azure.upbound.io/v1beta1"),
								Kind:       ptr.To("ResourceGroup"),
								Spec: &azv1beta1.ResourceGroupSpec{
									ForProvider: &azv1beta1.ResourceGroupSpecForProvider{
										Location: ptr.To("us-east-1"),
									},
								},
							}),
							"acct": toResource(&storagev1beta1.Account{
								APIVersion: ptr.To("storage.azure.upbound.io/v1beta1"),
								Kind:       ptr.To("Account"),
								Metadata: &metav1.ObjectMeta{
									Annotations: &map[string]string{
										"crossplane.io/external-name": "examplexr",
									},
								},
								Spec: &storagev1beta1.AccountSpec{
									ForProvider: &storagev1beta1.AccountSpecForProvider{
										ResourceGroupName:               ptr.To("super-group"),
										AccountTier:                     ptr.To("Standard"),
										AccountReplicationType:          ptr.To("LRS"),
										Location:                        ptr.To("us-east-1"),
										InfrastructureEncryptionEnabled: ptr.To(true),
										BlobProperties: &[]storagev1beta1.AccountSpecForProviderBlobPropertiesItem{{
											VersioningEnabled: ptr.To(true),
										}},
									},
								},
							}),
							"cont": toResource(&storagev1beta1.Container{
								APIVersion: ptr.To("storage.azure.upbound.io/v1beta1"),
								Kind:       ptr.To("Container"),
								Spec: &storagev1beta1.ContainerSpec{
									ForProvider: &storagev1beta1.ContainerSpecForProvider{
										StorageAccountName:  ptr.To("examplexr"),
										ContainerAccessType: ptr.To("private"),
									},
								},
							}),
						},
					},
				},
			},
		},
		"ResourceGroupCreatedWithPublicACL": {
			reason: "If the resource group has been created with public ACL requested, all resources should be desired.",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Observed: &fnv1.State{
						Composite: toResource(&v1alpha1.XStorageBucket{
							Metadata: &metav1.ObjectMeta{
								Name: ptr.To("example-xr"),
							},
							Spec: &v1alpha1.XStorageBucketSpec{
								Parameters: &v1alpha1.XStorageBucketSpecParameters{
									Location:   ptr.To("us-east-1"),
									ACL:        ptr.To("public"),
									Versioning: ptr.To(false),
								},
							},
						}),
						Resources: map[string]*fnv1.Resource{
							"rg": toResource(&azv1beta1.ResourceGroup{
								APIVersion: ptr.To("azure.upbound.io/v1beta1"),
								Kind:       ptr.To("ResourceGroup"),
								Metadata: &metav1.ObjectMeta{
									Annotations: &map[string]string{
										"crossplane.io/external-name": "super-group",
									},
								},
								Spec: &azv1beta1.ResourceGroupSpec{
									ForProvider: &azv1beta1.ResourceGroupSpecForProvider{
										Location: ptr.To("us-east-1"),
									},
								},
							}),
						},
					},
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta:    &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{},
					Desired: &fnv1.State{
						Resources: map[string]*fnv1.Resource{
							"rg": toResource(&azv1beta1.ResourceGroup{
								APIVersion: ptr.To("azure.upbound.io/v1beta1"),
								Kind:       ptr.To("ResourceGroup"),
								Spec: &azv1beta1.ResourceGroupSpec{
									ForProvider: &azv1beta1.ResourceGroupSpecForProvider{
										Location: ptr.To("us-east-1"),
									},
								},
							}),
							"acct": toResource(&storagev1beta1.Account{
								APIVersion: ptr.To("storage.azure.upbound.io/v1beta1"),
								Kind:       ptr.To("Account"),
								Metadata: &metav1.ObjectMeta{
									Annotations: &map[string]string{
										"crossplane.io/external-name": "examplexr",
									},
								},
								Spec: &storagev1beta1.AccountSpec{
									ForProvider: &storagev1beta1.AccountSpecForProvider{
										ResourceGroupName:               ptr.To("super-group"),
										AccountTier:                     ptr.To("Standard"),
										AccountReplicationType:          ptr.To("LRS"),
										Location:                        ptr.To("us-east-1"),
										InfrastructureEncryptionEnabled: ptr.To(true),
										BlobProperties: &[]storagev1beta1.AccountSpecForProviderBlobPropertiesItem{{
											VersioningEnabled: ptr.To(false),
										}},
									},
								},
							}),
							"cont": toResource(&storagev1beta1.Container{
								APIVersion: ptr.To("storage.azure.upbound.io/v1beta1"),
								Kind:       ptr.To("Container"),
								Spec: &storagev1beta1.ContainerSpec{
									ForProvider: &storagev1beta1.ContainerSpecForProvider{
										StorageAccountName:  ptr.To("examplexr"),
										ContainerAccessType: ptr.To("blob"),
									},
								},
							}),
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := &Function{log: logging.NewNopLogger()}
			rsp, err := f.RunFunction(tc.args.ctx, tc.args.req)

			if diff := cmp.Diff(tc.want.rsp, rsp, protocmp.Transform()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want rsp, +got rsp:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}

func toResource(in any) *fnv1.Resource {
	obj := composite.New()
	_ = convertViaJSON(obj, in)
	pb, _ := resource.AsStruct(obj)
	return &fnv1.Resource{
		Resource: pb,
	}
}
