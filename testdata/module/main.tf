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
}

resource "azurerm_virtual_network" "test2" {
  name                = "bar"
  resource_group_name = data.azurerm_resource_group.test.name
  location            = azurerm_virtual_network.test.location
  address_space       = ["10.0.0.0/16"]
}

locals {
  vnet_location = azurerm_virtual_network.test.location
}

module "test" {
  source   = "./submodule"
  location = azurerm_virtual_network.test2.location
}
