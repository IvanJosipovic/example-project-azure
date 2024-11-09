from crossplane.function import resource
from crossplane.function.proto.v1 import run_function_pb2 as fnv1

from .model.io.k8s.apimachinery.pkg.apis.meta import v1 as metav1
from .model.io.upbound.azure.resourcegroup import v1beta1 as rgv1beta1
from .model.io.upbound.azure.storage.account import v1beta1 as acctv1beta1
from .model.io.upbound.azure.storage.container import v1beta1 as contv1beta1
from .model.com.example.platform.xstoragebucket import v1alpha1


def compose(req: fnv1.RunFunctionRequest, rsp: fnv1.RunFunctionResponse):
    observed_xr = v1alpha1.XStorageBucket(**req.observed.composite.resource)
    params = observed_xr.spec.parameters

    desired_group = rgv1beta1.ResourceGroup(
        apiVersion="azure.upbound.io/v1beta1",
        kind="ResourceGroup",
        spec=rgv1beta1.Spec(
            forProvider=rgv1beta1.ForProvider(
                location=params.location,
            ),
        ),
    )
    resource.update(rsp.desired.resources["group"], desired_group)

    # Return early if Crossplane hasn't observed the group yet. This means it
    # hasn't been created yet. This function will be called again after it is.
    # We want the group to be created so we can refer to its external name.
    if "group" not in req.observed.resources:
        return

    observed_group = acctv1beta1.Account(**req.observed.resources["group"].resource)
    if observed_group.metadata is None or observed_group.metadata.annotations is None:
        return
    if "crossplane.io/external-name" not in observed_group.metadata.annotations:
        return

    group_external_name = observed_group.metadata.annotations[
        "crossplane.io/external-name"
    ]

    # Storage account names must be 3-24 character, lowercase alphanumeric
    # strings that are globally unique within Azure. We try to generate a valid
    # one automatically by deriving it from the XR name, which should always be
    # alphanumeric, lowercase, and separated by hyphens.
    account_external_name = observed_xr.metadata.name.replace("-", "")  # type: ignore  # Name is an optional field, but it'll always be set.

    desired_acct = acctv1beta1.Account(
        apiVersion="storage.azure.upbound.io/v1beta1",
        kind="Account",
        metadata=metav1.ObjectMeta(
            annotations={
                "crossplane.io/external-name": account_external_name,
            },
        ),
        spec=acctv1beta1.Spec(
            forProvider=acctv1beta1.ForProvider(
                resourceGroupName=group_external_name,
                accountTier="Standard",
                accountReplicationType="LRS",
                location=params.location,
                infrastructureEncryptionEnabled=True,
                blobProperties=[
                    acctv1beta1.BlobProperty(
                        versioningEnabled=params.versioning,
                    ),
                ],
            ),
        ),
    )
    resource.update(rsp.desired.resources["acct"], desired_acct)

    desired_cont = contv1beta1.Container(
        apiVersion="storage.azure.upbound.io/v1beta1",
        kind="Container",
        spec=contv1beta1.Spec(
            forProvider=contv1beta1.ForProvider(
                storageAccountName=account_external_name,
                containerAccessType="blob" if params.acl == "public" else "private",
            ),
        ),
    )
    resource.update(rsp.desired.resources["cont"], desired_cont)
