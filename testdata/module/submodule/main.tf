variable "resource_group_name" {
  type = string
}

data "azurerm_resource_group" "test" {
  name = var.resource_group_name
}


# This resource isn't in the TF state
resource "azurerm_virtual_network" "test" {
  name                = "baz"
  resource_group_name = data.azurerm_resource_group.test.name
  location            = data.azurerm_resource_group.test.location
  address_space       = ["10.0.0.0/16"]
}
