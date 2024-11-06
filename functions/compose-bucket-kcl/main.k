import models.io.upbound.azure.storage.v1beta1

oxr = option("params").oxr # observed composite resource
params = oxr.spec.parameters

containerAccessType = "blob" if params.acl == "public" else "private"
accountName = "{}-account".format(oxr.metadata.name)

_metadata = lambda name: str -> any {
  {
    name = name
    annotations = {
      "krm.kcl.dev/composition-resource-name" = name
    }
  }
}

_items = [
    v1beta1.Account{
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
            }
        }
    },
    v1beta1.Container{
        metadata: _metadata("{}-container".format(oxr.metadata.name))
        spec = {
            forProvider = {
                containerAccessType = containerAccessType
            }
        }
    }
]
items = _items
