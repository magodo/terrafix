variable "name" {
  type = string
}

variable "location" {
  type = string
}

resource "azurerm_resource_group" "test" {
  name     = var.name
  location = var.location
}
