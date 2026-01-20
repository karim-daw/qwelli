# Same PostgreSQL configuration as before but optimized for AKS
resource "azurerm_postgresql_flexible_server" "qwelli" {
  name                   = "${var.prefix}-postgres-${random_string.suffix.result}"
  resource_group_name    = azurerm_resource_group.qwelli.name
  location               = azurerm_resource_group.qwelli.location
  version                = "16"
  administrator_login    = var.db_admin_username
  administrator_password = var.db_password

  storage_mb            = var.db_storage_mb
  backup_retention_days = var.db_backup_retention_days
  geo_redundant_backup_enabled = var.environment == "production" ? true : false

  sku_name = var.db_sku_name

  # High availability for production
  dynamic "high_availability" {
    for_each = var.environment == "production" ? [1] : []
    content {
      mode                      = "ZoneRedundant"
      standby_availability_zone = "2"
    }
  }

  # VNet integration - Private access only
  delegated_subnet_id = azurerm_subnet.database.id
  private_dns_zone_id = azurerm_private_dns_zone.postgres.id

  maintenance_window {
    day_of_week  = 0
    start_hour   = 2
    start_minute = 0
  }

  tags = var.tags

  depends_on = [azurerm_private_dns_zone_virtual_network_link.postgres]
}

# Enable pgvector extension
resource "azurerm_postgresql_flexible_server_configuration" "azure_extensions" {
  name      = "azure.extensions"
  server_id = azurerm_postgresql_flexible_server.qwelli.id
  value     = "vector,pg_stat_statements"
}

# Performance tuning for vector workloads
resource "azurerm_postgresql_flexible_server_configuration" "shared_buffers" {
  name      = "shared_buffers"
  server_id = azurerm_postgresql_flexible_server.qwelli.id
  value     = var.environment == "production" ? "2GB" : "256MB"
}

resource "azurerm_postgresql_flexible_server_configuration" "effective_cache_size" {
  name      = "effective_cache_size"
  server_id = azurerm_postgresql_flexible_server.qwelli.id
  value     = var.environment == "production" ? "6GB" : "512MB"
}

resource "azurerm_postgresql_flexible_server_configuration" "maintenance_work_mem" {
  name      = "maintenance_work_mem"
  server_id = azurerm_postgresql_flexible_server.qwelli.id
  value     = var.environment == "production" ? "512MB" : "64MB"
}

resource "azurerm_postgresql_flexible_server_configuration" "max_connections" {
  name      = "max_connections"
  server_id = azurerm_postgresql_flexible_server.qwelli.id
  value     = var.db_max_connections
}

resource "azurerm_postgresql_flexible_server_configuration" "work_mem" {
  name      = "work_mem"
  server_id = azurerm_postgresql_flexible_server.qwelli.id
  value     = var.environment == "production" ? "16MB" : "4MB"
}

# Connection pooling settings
resource "azurerm_postgresql_flexible_server_configuration" "idle_in_transaction_timeout" {
  name      = "idle_in_transaction_session_timeout"
  server_id = azurerm_postgresql_flexible_server.qwelli.id
  value     = "300000"  # 5 minutes
}

# Create database
resource "azurerm_postgresql_flexible_server_database" "qwelli" {
  name      = var.db_name
  server_id = azurerm_postgresql_flexible_server.qwelli.id
  charset   = "UTF8"
  collation = "en_US.utf8"
}
