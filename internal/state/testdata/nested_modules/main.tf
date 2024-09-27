provider "azurerm" {
  features {
    resource_group {
      prevent_deletion_if_contains_resources = false
    }
  }
}

resource "azurerm_resource_group" "test" {
  count    = 2
  name     = "terrafix-test-${count.index}"
  location = "westus2"
}

data "azurerm_resource_group" "test" {
  name = azurerm_resource_group.test[0].name
}

module "test" {
  source = "./module"
}
