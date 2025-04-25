from .model.io.upbound.dev.meta.compositiontest import v1alpha1 as compositiontest
from .model.io.k8s.apimachinery.pkg.apis.meta import v1 as k8s
from .model.io.upbound.azure.resourcegroup import v1beta1 as rgv1beta1
from .model.io.upbound.azure.storage.account import v1beta1 as acctv1beta1
from .model.io.upbound.azure.storage.container import v1beta1 as contv1beta1
from .model.com.example.platform.xstoragebucket import v1alpha1 as platformv1alpha1

xStorageBucket = platformv1alpha1.XStorageBucket(
    apiVersion="platform.example.com/v1alpha1",
    kind="XStorageBucket",
    metadata=k8s.ObjectMeta(
        name="example-python"
    ),
    spec = platformv1alpha1.Spec(
        compositionSelector=platformv1alpha1.CompositionSelector(
            matchLabels={
                "language": "python",
            },
        ),
        parameters = platformv1alpha1.Parameters(
            acl="public",
            location="eastus",
            versioning=True,
        ),
    ),
)

group = rgv1beta1.ResourceGroup(
    apiVersion="azure.upbound.io/v1beta1",
    kind="ResourceGroup",
    spec=rgv1beta1.Spec(
        forProvider=rgv1beta1.ForProvider(
            location="eastus",
        )
    )
)

acct = acctv1beta1.Account(
    apiVersion="storage.azure.upbound.io/v1beta1",
    kind="Account",
    spec=acctv1beta1.Spec(
        forProvider=acctv1beta1.ForProvider(
            resourceGroupName="group-name",
            accountTier="Standard",
            accountReplicationType="LRS",
            location="eastus",
            infrastructureEncryptionEnabled=True,
            blobProperties=[
                acctv1beta1.BlobProperty(
                    versioningEnabled=False,
                ),
            ],
        ),
    )
)

cont = contv1beta1.Container(
    apiVersion="storage.azure.upbound.io/v1beta1",
    kind="Container",
    spec=contv1beta1.Spec(
        forProvider=contv1beta1.ForProvider(
            storageAccountName="acct-name",
        )
    )
)

test = compositiontest.CompositionTest(
    metadata=k8s.ObjectMeta(
        name="test-xstoragebucket-python",
    ),
    spec = compositiontest.Spec(
        assertResources=[
            xStorageBucket.model_dump(exclude_unset=True),
            group.model_dump(exclude_unset=True),
            # TODO: Assert other resources. This is tricker for Python than KCL
            # since we let Crossplane name our resources.
        ],
        compositionPath="apis/python/composition.yaml",
        xrPath="examples/python/example.yaml",
        xrdPath="apis/xstoragebuckets/definition.yaml",
        timeoutSeconds=120,
        validate=False,
    )
)
