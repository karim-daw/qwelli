# Azure Container Registry for Docker images
resource "azurerm_container_registry" "qwelli" {
  name                = "${var.prefix}acr${random_string.suffix.result}"
  resource_group_name = azurerm_resource_group.qwelli.name
  location            = azurerm_resource_group.qwelli.location
  sku                 = var.acr_sku
  admin_enabled       = false  # Use managed identity instead

  # Public network access
  public_network_access_enabled = var.environment == "production" ? false : true

  # Network rule set for production
  dynamic "network_rule_set" {
    for_each = var.environment == "production" ? [1] : []
    content {
      default_action = "Deny"

      ip_rule {
        action   = "Allow"
        ip_range = var.allowed_ip_range
      }
    }
  }

  # Enable zone redundancy for Premium SKU in production
  zone_redundancy_enabled = var.acr_sku == "Premium" && var.environment == "production" ? true : false

  tags = var.tags
}

# Grant Container Instance identity pull access to ACR
resource "azurerm_role_assignment" "acr_pull" {
  scope                = azurerm_container_registry.qwelli.id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_user_assigned_identity.qwelli.principal_id
}

# Private endpoint for ACR (Premium SKU only)
resource "azurerm_private_endpoint" "acr" {
  count               = var.acr_sku == "Premium" && var.environment == "production" ? 1 : 0
  name                = "${var.prefix}-acr-pe"
  location            = azurerm_resource_group.qwelli.location
  resource_group_name = azurerm_resource_group.qwelli.name
  subnet_id           = azurerm_subnet.database.id

  private_service_connection {
    name                           = "${var.prefix}-acr-psc"
    private_connection_resource_id = azurerm_container_registry.qwelli.id
    is_manual_connection           = false
    subresource_names              = ["registry"]
  }

  private_dns_zone_group {
    name                 = "acr-dns-zone-group"
    private_dns_zone_ids = [azurerm_private_dns_zone.acr[0].id]
  }

  tags = var.tags
}

# Private DNS Zone for ACR
resource "azurerm_private_dns_zone" "acr" {
  count               = var.acr_sku == "Premium" && var.environment == "production" ? 1 : 0
  name                = "privatelink.azurecr.io"
  resource_group_name = azurerm_resource_group.qwelli.name

  tags = var.tags
}

# Link ACR Private DNS Zone to VNet
resource "azurerm_private_dns_zone_virtual_network_link" "acr" {
  count                 = var.acr_sku == "Premium" && var.environment == "production" ? 1 : 0
  name                  = "${var.prefix}-acr-dns-link"
  resource_group_name   = azurerm_resource_group.qwelli.name
  private_dns_zone_name = azurerm_private_dns_zone.acr[0].name
  virtual_network_id    = azurerm_virtual_network.qwelli.id

  tags = var.tags
}
