import models.com.example.platform.v1alpha1 as platformv1alpha1
import models.io.upbound.azure.v1beta1 as azurev1beta1
import models.io.upbound.azure.storage.v1beta1 as storagev1beta1
import models.io.upbound.dev.meta.v1alpha1 as metav1alpha1

_items = [
    metav1alpha1.CompositionTest{
        metadata.name="test-xstoragebucket-kcl"
        spec= {
            assertResources: [
                platformv1alpha1.XStorageBucket{
                    metadata.name: "example-kcl"
                    spec: {
                        parameters: {
                            acl: "public"
                            location: "eastus"
                            versioning: True
                        }
                        compositionSelector:{
                            matchLabels: {
                                language: "kcl"
                            }
                        }
                    }
                }
                azurev1beta1.ResourceGroup{
                    metadata.name: "example-kcl-group"
                    spec.forProvider: {
                        location: "eastus"
                    }
                }
                storagev1beta1.Account{
                    metadata.name: "examplekcl"
                    spec.forProvider: {
                        accountTier: "Standard"
                        accountReplicationType: "LRS"
                        location: "eastus"
                        blobProperties = [
                            {
                                versioningEnabled = True
                            }
                        ]
                        infrastructureEncryptionEnabled = True
                        resourceGroupNameRef = {
                            name = "example-kcl-group"
                        }
                    }
                }
                storagev1beta1.Container{
                    metadata.name: "example-kcl-container"
                    spec.forProvider: {
                        containerAccessType: "blob"
                        storageAccountNameRef: {
                            name: "examplekcl"
                        }
                    }
                }
            ]
            compositionPath: "apis/kcl/composition.yaml"
            xrPath: "examples/kcl/example.yaml"
            xrdPath: "apis/xstoragebuckets/definition.yaml"
            timeoutSeconds: 120
            validate: False
        }
    }
]
items = _items
