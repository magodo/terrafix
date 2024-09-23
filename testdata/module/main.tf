provider "azurerm" {
  features {}
}

data "azurerm_resource_group" "test" {
  name = "foo"
}

resource "azurerm_virtual_network" "test" {
  name                = "foo"
  resource_group_name = data.azurerm_resource_group.test.name
  location            = "westus2"
  address_space       = ["10.0.0.0/16"]
  encryption {
    enforcement = "AllowUnencrypted"
  }
}

resource "azurerm_virtual_network" "test2" {
  name                = "bar"
  resource_group_name = data.azurerm_resource_group.test.name
  location            = azurerm_virtual_network.test.location
  address_space       = ["10.0.0.0/16"]
  encryption {
    enforcement = "AllowUnencrypted"
  }
}

locals {
  ds_attribute_ref    = data.azurerm_resource_group.test.id
  attribute_ref       = azurerm_virtual_network.test.name
  ro_attribute_ref    = azurerm_virtual_network.test.id
  block_attribute_ref = azurerm_virtual_network.test.encryption[0].enforcement

  vnet_location = azurerm_virtual_network.test.location
}
