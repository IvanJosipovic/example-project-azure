package main

import (
	"context"
	"encoding/json"
	"strings"

	"dev.upbound.io/models/com/example/platform/v1alpha1"
	metav1 "dev.upbound.io/models/io/k8s/meta/v1"
	storagev1beta1 "dev.upbound.io/models/io/upbound/azure/storage/v1beta1"
	azv1beta1 "dev.upbound.io/models/io/upbound/azure/v1beta1"
	"k8s.io/utils/ptr"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/crossplane/function-sdk-go/response"
)

// Function is your composition function.
type Function struct {
	fnv1.UnimplementedFunctionRunnerServiceServer

	log logging.Logger
}

// RunFunction runs the Function.
func (f *Function) RunFunction(_ context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
	f.log.Info("Running function", "tag", req.GetMeta().GetTag())
	rsp := response.To(req, response.DefaultTTL)

	observedComposite, err := request.GetObservedCompositeResource(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot get xr"))
		return rsp, nil
	}

	observedComposed, err := request.GetObservedComposedResources(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot get observed resources"))
		return rsp, nil
	}

	var xr v1alpha1.XStorageBucket
	if err := convertViaJSON(&xr, observedComposite.Resource); err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot convert xr"))
		return rsp, nil
	}

	params := xr.Spec.Parameters
	if ptr.Deref(params.Location, "") == "" {
		response.Fatal(rsp, errors.Wrap(err, "missing location"))
		return rsp, nil
	}

	// We'll collect our desired composed resources into this map, then convert
	// them to the SDK's types and set them in the response when we return.
	desiredComposed := make(map[resource.Name]any)
	defer func() {
		desiredComposedResources, err := request.GetDesiredComposedResources(req)
		if err != nil {
			response.Fatal(rsp, errors.Wrap(err, "cannot get desired resources"))
			return
		}

		for name, obj := range desiredComposed {
			c := composed.New()
			if err := convertViaJSON(c, obj); err != nil {
				response.Fatal(rsp, errors.Wrapf(err, "cannot convert %s to unstructured", name))
				return
			}
			desiredComposedResources[name] = &resource.DesiredComposed{Resource: c}
		}

		if err := response.SetDesiredComposedResources(rsp, desiredComposedResources); err != nil {
			response.Fatal(rsp, errors.Wrap(err, "cannot set desired resources"))
			return
		}
	}()

	resourceGroup := &azv1beta1.ResourceGroup{
		APIVersion: ptr.To("azure.upbound.io/v1beta1"),
		Kind:       ptr.To("ResourceGroup"),
		Spec: &azv1beta1.ResourceGroupSpec{
			ForProvider: &azv1beta1.ResourceGroupSpecForProvider{
				Location: params.Location,
			},
		},
	}
	desiredComposed["rg"] = resourceGroup

	// Return early if Crossplane hasn't observed the resource group yet. This
	// means it hasn't been created yet. This function will be called again
	// after it is.
	observedResourceGroup, ok := observedComposed["rg"]
	if !ok {
		response.Normal(rsp, "waiting for resource group to be created").TargetCompositeAndClaim()
		return rsp, nil
	}

	// The desired account needs to refer to the resource group by its external
	// name, which is stored in its external name annotation. Return early if
	// the ResourceGroup's external-name annotation isn't set yet.
	rgExternalName := observedResourceGroup.Resource.GetAnnotations()["crossplane.io/external-name"]
	if rgExternalName == "" {
		response.Normal(rsp, "waiting for resource group to be created").TargetCompositeAndClaim()
		return rsp, nil
	}

	// Storage account names must be 3-24 character, lowercase alphanumeric
	// strings that are globally unique within Azure. We try to generate a valid
	// one automatically by deriving it from the XR name, which should always be
	// alphanumeric, lowercase, and separated by hyphens.
	acctExternalName := strings.ReplaceAll(*xr.Metadata.Name, "-", "")

	acct := &storagev1beta1.Account{
		APIVersion: ptr.To("storage.azure.upbound.io/v1beta1"),
		Kind:       ptr.To("Account"),
		Metadata: &metav1.ObjectMeta{
			Annotations: &map[string]string{
				"crossplane.io/external-name": acctExternalName,
			},
		},
		Spec: &storagev1beta1.AccountSpec{
			ForProvider: &storagev1beta1.AccountSpecForProvider{
				ResourceGroupName:               &rgExternalName,
				AccountTier:                     ptr.To("Standard"),
				AccountReplicationType:          ptr.To("LRS"),
				Location:                        params.Location,
				InfrastructureEncryptionEnabled: ptr.To(true),
				BlobProperties: &[]storagev1beta1.AccountSpecForProviderBlobPropertiesItem{{
					VersioningEnabled: params.Versioning,
				}},
			},
		},
	}
	desiredComposed["acct"] = acct

	cont := &storagev1beta1.Container{
		APIVersion: ptr.To("storage.azure.upbound.io/v1beta1"),
		Kind:       ptr.To("Container"),
		Spec: &storagev1beta1.ContainerSpec{
			ForProvider: &storagev1beta1.ContainerSpecForProvider{
				StorageAccountName: &acctExternalName,
			},
		},
	}
	if ptr.Deref(params.ACL, "") == "public" {
		cont.Spec.ForProvider.ContainerAccessType = ptr.To("blob")
	} else {
		cont.Spec.ForProvider.ContainerAccessType = ptr.To("private")
	}
	desiredComposed["cont"] = cont

	return rsp, nil
}

func convertViaJSON(to, from any) error {
	bs, err := json.Marshal(from)
	if err != nil {
		return err
	}
	return json.Unmarshal(bs, to)
}
