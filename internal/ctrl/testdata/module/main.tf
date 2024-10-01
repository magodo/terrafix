provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "terrafix-ctrl"
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

resource "azurerm_container_registry" "test" {
  name                = "terrafixacr"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku                 = "Basic"
}

locals {
  vnet_location = azurerm_virtual_network.test.location
  vnet_guid     = azurerm_virtual_network.test.guid
}

module "test" {
  source              = "./submodule"
  resource_group_name = azurerm_resource_group.test.name
  depends_on          = [azurerm_resource_group.test]
}
