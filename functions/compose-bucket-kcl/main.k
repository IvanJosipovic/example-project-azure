import models.io.upbound.azure.v1beta1 as azurev1beta1
import models.io.upbound.azure.storage.v1beta1 as storagev1beta1

oxr = option("params").oxr # observed composite resource
params = oxr.spec.parameters

containerAccessType = "blob" if params.acl == "public" else "private"
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
                location = params.location
            }
        }
    },
    storagev1beta1.Account{
        metadata = _metadata(accountName)
        spec = {
            forProvider = {
                accountTier = "Standard"
                accountReplicationType = "LRS"
                location = params.location
                blobProperties = [
                    {
                        versioningEnabled = params.versioning
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
