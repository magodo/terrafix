provider "azurerm" {
  features {}
}

resource "azurerm_virtual_network" "test" {
  name                = "foo"
  resource_group_name = "foo"
  location            = "westus2"
  address_space       = ["10.0.0.0/16"]
  encryption {
    enforcement = "AllowUnencrypted"
  }
}

locals {
  attribute_ref       = azurerm_virtual_network.test.name
  ro_attribute_ref    = azurerm_virtual_network.test.id
  block_attribute_ref = azurerm_virtual_network.test.encryption.0.enforcement
}
