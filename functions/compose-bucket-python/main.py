from crossplane.function import resource
from crossplane.function.proto.v1 import run_function_pb2 as fnv1

from .model.io.upbound.azure.storage.account import v1beta1 as acctv1beta1
from .model.io.upbound.azure.storage.container import v1beta1 as contv1beta1
from .model.com.example.platform.xstoragebucket import v1alpha1


def compose(req: fnv1.RunFunctionRequest, rsp: fnv1.RunFunctionResponse):
    observed_xr = v1alpha1.XStorageBucket(**req.observed.composite.resource)
    params = observed_xr.spec.parameters

    desired_acct = acctv1beta1.Account(
        apiVersion="storage.azure.upbound.io/v1beta1",
        kind="Account",
        spec=acctv1beta1.Spec(
            forProvider=acctv1beta1.ForProvider(
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

    # Return early if Crossplane hasn't observed the account yet. This means it
    # hasn't been created yet. This function will be called again after it is.
    # We want the account to be created so we can refer to its external name.
    if "account" not in req.observed.resources:
        return

    observed_acct = acctv1beta1.Account(**req.observed.resources["account"].resource)
    if observed_acct.metadata is None or observed_acct.metadata.annotations is None:
        return
    if "crossplane.io/external-name" not in observed_acct.metadata.annotations:
        return

    account_external_name = observed_acct.metadata.annotations[
        "crossplane.io/external-name"
    ]

    desired_cont = contv1beta1.Container(
        apiVersion="storage.azure.upbound.io/v1beta1",
        kind="Container",
        spec=contv1beta1.Spec(
            forProvider=contv1beta1.ForProvider(
                containerAccessType="blob" if params.acl == "public" else "private",
                storageAccountName=account_external_name,
            ),
        ),
    )
    resource.update(rsp.desired.resources["cont"], desired_cont)
