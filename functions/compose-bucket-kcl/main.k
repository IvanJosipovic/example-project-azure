import models.io.upbound.azure.v1beta1 as azurev1beta1
import models.io.upbound.azure.storage.v1beta1 as storagev1beta1
import models.com.example.platform.v1alpha1.xstoragebucket as xstoragebucketv1alpha1

oxr = xstoragebucketv1alpha1.XStorageBucket{**option("params").oxr} # observed composite resource

containerAccessType = "blob" if oxr.spec.parameters.acl == "public" else "private"
groupName = "{}-group".format(oxr.metadata.name)
accountName = oxr.metadata.name.replace("-", "")

_metadata = lambda name: str -> any {
  {
    name = name
    annotations = {
      "krm.kcl.dev/composition-resource-name" = name
    }
  }
}

_items = [
    azurev1beta1.ResourceGroup{
        metadata = _metadata(groupName)
        spec = {
            forProvider = {
                location = oxr.spec.parameters.location
            }
        }
    },
    storagev1beta1.Account{
        metadata = _metadata(accountName)
        spec = {
            forProvider = {
                accountTier = "Standard"
                accountReplicationType = "LRS"
                location = oxr.spec.parameters.location
                blobProperties = [
                    {
                        versioningEnabled = oxr.spec.parameters.versioning
                    }
                ]
                infrastructureEncryptionEnabled = True
                resourceGroupNameRef = {
                    name = groupName
                }
            }
        }
    },
    storagev1beta1.Container{
        metadata: _metadata("{}-container".format(oxr.metadata.name))
        spec = {
            forProvider = {
                containerAccessType = containerAccessType
                storageAccountNameRef = {
                    name = accountName
                }
            }
        }
    }
]
items = _items
