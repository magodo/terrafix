resource "azurerm_resource_group" "test" {
  name     = "terrafix-test-mod"
  location = "westus2"
}

data "azurerm_resource_group" "test" {
  name = azurerm_resource_group.test.name
}

module "test" {
  source = "./module"
}
