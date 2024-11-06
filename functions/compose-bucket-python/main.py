from crossplane.function import resource
from crossplane.function.proto.v1 import run_function_pb2 as fnv1
from model.io.k8s.apimachinery.pkg.apis.meta import v1 as metav1
from model.io.upbound.azure.storage.account import v1beta1 as acctv1beta1
from model.io.upbound.azure.storage.container import v1beta1 as contv1beta1
from model.com.example.platform.xstoragebucket import v1alpha1

def compose(req: fnv1.RunFunctionRequest, rsp: fnv1.RunFunctionResponse):
    observed_xr = v1alpha1.XStorageBucket(**req.observed.composite.resource)
    xr_name = observed_xr.metadata.name
    acct_name = xr_name + "-account"
    params = observed_xr.spec.parameters

    acct = acctv1beta1.Account(
        apiVersion="storage.azure.upbound.io/v1beta1",
        kind="Account",
        metadata=metav1.ObjectMeta(
            name=acct_name,
        ),
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
    resource.update(rsp.desired.resources[acct.metadata.name], acct)

    accessType = "blob" if params.acl == "public" else "private"
    cont = contv1beta1.Container(
        apiVersion="storage.azure.upbound.io/v1beta1",
        kind="Container",
        metadata=metav1.ObjectMeta(
            name=xr_name + "-container",
        ),
        spec=contv1beta1.Spec(
            forProvider=contv1beta1.ForProvider(
                containerAccessType=accessType,
            ),
        ),
    )
    resource.update(rsp.desired.resources[cont.metadata.name], cont)
