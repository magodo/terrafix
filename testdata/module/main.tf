provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "terrafix-magodo"
  location = "westus2"
}

resource "azurerm_virtual_network" "test" {
  name                = "foo"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  address_space       = ["10.0.0.0/16"]
}

resource "azurerm_virtual_network" "test2" {
  name                = "bar"
  resource_group_name = azurerm_virtual_network.test.resource_group_name
  location            = azurerm_virtual_network.test.location
  address_space       = ["10.0.0.0/16"]
}

locals {
  vnet_location = azurerm_virtual_network.test.location
}

module "test" {
  source              = "./submodule"
  resource_group_name = azurerm_resource_group.test.name
  depends_on          = [azurerm_resource_group.test]
}
