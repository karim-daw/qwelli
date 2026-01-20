# Azure Key Vault for secure secret storage
resource "azurerm_key_vault" "qwelli" {
  name                        = "${var.prefix}-kv-${random_string.suffix.result}"
  location                    = azurerm_resource_group.qwelli.location
  resource_group_name         = azurerm_resource_group.qwelli.name
  enabled_for_disk_encryption = true
  tenant_id                   = data.azurerm_client_config.current.tenant_id
  soft_delete_retention_days  = 7
  purge_protection_enabled    = var.environment == "production" ? true : false
  sku_name                    = "standard"

  # Enable firewall
  network_acls {
    bypass         = "AzureServices"
    default_action = var.environment == "production" ? "Deny" : "Allow"

    # Allow specific IP ranges in production
    ip_rules = var.environment == "production" ? split(",", var.allowed_ip_range) : []
  }

  tags = var.tags
}

# Access policy for current user/service principal
resource "azurerm_key_vault_access_policy" "terraform" {
  key_vault_id = azurerm_key_vault.qwelli.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = data.azurerm_client_config.current.object_id

  secret_permissions = [
    "Get",
    "List",
    "Set",
    "Delete",
    "Recover",
    "Backup",
    "Restore",
    "Purge"
  ]

  certificate_permissions = [
    "Get",
    "List",
    "Create",
    "Delete"
  ]
}

# Store database password in Key Vault
resource "azurerm_key_vault_secret" "db_password" {
  name         = "db-password"
  value        = var.db_password
  key_vault_id = azurerm_key_vault.qwelli.id

  depends_on = [azurerm_key_vault_access_policy.terraform]

  tags = var.tags
}

# Store Voyage API key in Key Vault
resource "azurerm_key_vault_secret" "voyage_api_key" {
  name         = "voyage-api-key"
  value        = var.voyage_api_key
  key_vault_id = azurerm_key_vault.qwelli.id

  depends_on = [azurerm_key_vault_access_policy.terraform]

  tags = var.tags
}

# Managed Identity for Container Instances
resource "azurerm_user_assigned_identity" "qwelli" {
  name                = "${var.prefix}-identity"
  resource_group_name = azurerm_resource_group.qwelli.name
  location            = azurerm_resource_group.qwelli.location

  tags = var.tags
}

# Grant Container Identity access to Key Vault
resource "azurerm_key_vault_access_policy" "container" {
  key_vault_id = azurerm_key_vault.qwelli.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = azurerm_user_assigned_identity.qwelli.principal_id

  secret_permissions = [
    "Get",
    "List"
  ]

  depends_on = [azurerm_key_vault_access_policy.terraform]
}
