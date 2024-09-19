provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "src" {
  name     = "my-rg"
  location = "westeurope"
}

resource "azurerm_resource_group" "dst" {
  name     = azurerm_resource_group.src.name
  location = azurerm_resource_group.src.location
  tags = {
    source_id = azurerm_resource_group.src.id
  }
}

module "local" {
  source   = "./local"
  name     = azurerm_resource_group.src.name
  location = azurerm_resource_group.src.location
}
