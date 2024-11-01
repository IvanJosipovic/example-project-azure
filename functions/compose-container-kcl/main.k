
import models.v1beta1 as v1beta1

oxr = option("params").oxr # observed composite resource

containerAccessType = "blob" if oxr.spec.acl == "public" else "private"
accountName = "{}-account".format(oxr.metadata.name)
_items = [
    v1beta1.Account{
        metadata.name = accountName
        spec = v1beta1.StorageAzureUpboundIoV1beta1AccountSpec{
            forProvider = v1beta1.StorageAzureUpboundIoV1beta1AccountSpecForProvider{
                accountTier = "Standard"
                accountReplicationType = "LRS"
                location = oxr.spec.location
                blobProperties = [
                    v1beta1.StorageAzureUpboundIoV1beta1AccountSpecForProviderBlobPropertiesItems0{
                        versioningEnabled = oxr.spec.versioning
                    }
                ]
                infrastructureEncryptionEnabled = True
            }
        }
    },
    v1beta1.Container{
        metadata.name = "{}-container".format(oxr.metadata.name)
        spec = v1beta1.StorageAzureUpboundIoV1beta1ContainerSpec{
            forProvider = v1beta1.StorageAzureUpboundIoV1beta1ContainerSpecForProvider{
                containerAccessType = containerAccessType
            }
        }
    }
]
items = _items
