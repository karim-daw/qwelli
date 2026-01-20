# Azure Database for PostgreSQL Flexible Server
resource "azurerm_postgresql_flexible_server" "qwelli" {
  name                   = "${var.prefix}-postgres-${random_string.suffix.result}"
  resource_group_name    = azurerm_resource_group.qwelli.name
  location               = azurerm_resource_group.qwelli.location
  version                = "16"
  administrator_login    = var.db_admin_username
  administrator_password = var.db_password

  # Storage configuration
  storage_mb            = var.db_storage_mb
  backup_retention_days = var.db_backup_retention_days
  geo_redundant_backup_enabled = var.environment == "production" ? true : false

  # SKU configuration
  sku_name = var.db_sku_name

  # High availability for production
  dynamic "high_availability" {
    for_each = var.environment == "production" ? [1] : []
    content {
      mode = "ZoneRedundant"
    }
  }

  # Network configuration - delegated subnet for private access
  delegated_subnet_id = azurerm_subnet.database.id
  private_dns_zone_id = azurerm_private_dns_zone.postgres.id

  # Maintenance window
  maintenance_window {
    day_of_week  = 0  # Sunday
    start_hour   = 2  # 2 AM
    start_minute = 0
  }

  tags = var.tags

  depends_on = [azurerm_private_dns_zone_virtual_network_link.postgres]
}

# PostgreSQL Configuration - Enable pgvector extension
resource "azurerm_postgresql_flexible_server_configuration" "azure_extensions" {
  name      = "azure.extensions"
  server_id = azurerm_postgresql_flexible_server.qwelli.id
  value     = "vector"
}

# PostgreSQL Configuration - Shared preload libraries
resource "azurerm_postgresql_flexible_server_configuration" "shared_preload_libraries" {
  name      = "shared_preload_libraries"
  server_id = azurerm_postgresql_flexible_server.qwelli.id
  value     = "pg_stat_statements"
}

# PostgreSQL Configuration - Max connections
resource "azurerm_postgresql_flexible_server_configuration" "max_connections" {
  name      = "max_connections"
  server_id = azurerm_postgresql_flexible_server.qwelli.id
  value     = var.db_max_connections
}

# PostgreSQL Configuration - Connection timeout
resource "azurerm_postgresql_flexible_server_configuration" "connection_timeout" {
  name      = "idle_in_transaction_session_timeout"
  server_id = azurerm_postgresql_flexible_server.qwelli.id
  value     = "300000"  # 5 minutes in milliseconds
}

# Create Qwelli database
resource "azurerm_postgresql_flexible_server_database" "qwelli" {
  name      = var.db_name
  server_id = azurerm_postgresql_flexible_server.qwelli.id
  charset   = "UTF8"
  collation = "en_US.utf8"
}

# Firewall rule to allow Azure services (if not using private endpoint exclusively)
resource "azurerm_postgresql_flexible_server_firewall_rule" "azure_services" {
  count            = var.allow_azure_services ? 1 : 0
  name             = "AllowAzureServices"
  server_id        = azurerm_postgresql_flexible_server.qwelli.id
  start_ip_address = "0.0.0.0"
  end_ip_address   = "0.0.0.0"
}

# Enable Microsoft Defender for Cloud (Advanced Threat Protection) for production
resource "azurerm_mssql_server_security_alert_policy" "qwelli" {
  count              = var.environment == "production" ? 1 : 0
  resource_group_name = azurerm_resource_group.qwelli.name
  server_name        = azurerm_postgresql_flexible_server.qwelli.name
  state              = "Enabled"
  retention_days     = 30

  email_account_admins = true
  email_addresses      = var.alert_email_addresses
}
