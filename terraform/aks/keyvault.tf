# Azure Key Vault with VNet integration
resource "azurerm_key_vault" "qwelli" {
  name                        = "${var.prefix}-kv-${random_string.suffix.result}"
  location                    = azurerm_resource_group.qwelli.location
  resource_group_name         = azurerm_resource_group.qwelli.name
  enabled_for_disk_encryption = true
  tenant_id                   = data.azurerm_client_config.current.tenant_id
  soft_delete_retention_days  = 7
  purge_protection_enabled    = var.environment == "production" ? true : false
  sku_name                    = "standard"

  # Enable RBAC for Key Vault access
  enable_rbac_authorization = true

  # Network rules - allow access from AKS subnet
  network_acls {
    bypass                     = "AzureServices"
    default_action             = var.environment == "production" ? "Deny" : "Allow"
    virtual_network_subnet_ids = [azurerm_subnet.aks_nodes.id]

    # Allow specific IPs in production
    ip_rules = var.environment == "production" ? split(",", var.allowed_ip_range) : []
  }

  tags = var.tags
}

# Grant current user access for Terraform
resource "azurerm_role_assignment" "kv_admin_terraform" {
  scope                = azurerm_key_vault.qwelli.id
  role_definition_name = "Key Vault Administrator"
  principal_id         = data.azurerm_client_config.current.object_id
}

# Store database password
resource "azurerm_key_vault_secret" "db_password" {
  name         = "db-password"
  value        = var.db_password
  key_vault_id = azurerm_key_vault.qwelli.id

  depends_on = [azurerm_role_assignment.kv_admin_terraform]

  tags = var.tags
}

# Store Voyage API key
resource "azurerm_key_vault_secret" "voyage_api_key" {
  name         = "voyage-api-key"
  value        = var.voyage_api_key
  key_vault_id = azurerm_key_vault.qwelli.id

  depends_on = [azurerm_role_assignment.kv_admin_terraform]

  tags = var.tags
}

# Workload identity for pods to access Key Vault
resource "azurerm_user_assigned_identity" "qwelli_pod" {
  name                = "${var.prefix}-pod-identity"
  resource_group_name = azurerm_resource_group.qwelli.name
  location            = azurerm_resource_group.qwelli.location

  tags = var.tags
}

# Grant pod identity access to Key Vault
resource "azurerm_role_assignment" "pod_kv_secrets_user" {
  scope                = azurerm_key_vault.qwelli.id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = azurerm_user_assigned_identity.qwelli_pod.principal_id
}

# Federated identity credential for workload identity
resource "azurerm_federated_identity_credential" "qwelli_pod" {
  name                = "${var.prefix}-federated-credential"
  resource_group_name = azurerm_resource_group.qwelli.name
  audience            = ["api://AzureADTokenExchange"]
  issuer              = azurerm_kubernetes_cluster.qwelli.oidc_issuer_url
  parent_id           = azurerm_user_assigned_identity.qwelli_pod.id
  subject             = "system:serviceaccount:${var.kubernetes_namespace}:qwelli-sa"
}
