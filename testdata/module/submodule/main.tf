variable "location" {
  type = string
}

resource "azurerm_virtual_network" "test" {
  name                = "baz"
  resource_group_name = "ababab"
  location            = var.location
  address_space       = ["10.0.0.0/16"]
}
