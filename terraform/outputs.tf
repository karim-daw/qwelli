# Resource Group
output "resource_group_name" {
  description = "Name of the resource group"
  value       = azurerm_resource_group.qwelli.name
}

output "resource_group_location" {
  description = "Location of the resource group"
  value       = azurerm_resource_group.qwelli.location
}

# Network
output "vnet_id" {
  description = "Virtual Network ID"
  value       = azurerm_virtual_network.qwelli.id
}

output "vnet_name" {
  description = "Virtual Network name"
  value       = azurerm_virtual_network.qwelli.name
}

# Database
output "postgres_fqdn" {
  description = "PostgreSQL server FQDN"
  value       = azurerm_postgresql_flexible_server.qwelli.fqdn
}

output "postgres_server_name" {
  description = "PostgreSQL server name"
  value       = azurerm_postgresql_flexible_server.qwelli.name
}

output "postgres_database_name" {
  description = "PostgreSQL database name"
  value       = azurerm_postgresql_flexible_server_database.qwelli.name
}

output "postgres_connection_string" {
  description = "PostgreSQL connection string (without password)"
  value       = "postgresql://${var.db_admin_username}@${azurerm_postgresql_flexible_server.qwelli.fqdn}:5432/${var.db_name}?sslmode=require"
  sensitive   = true
}

# Container Registry
output "acr_login_server" {
  description = "Container registry login server"
  value       = azurerm_container_registry.qwelli.login_server
}

output "acr_name" {
  description = "Container registry name"
  value       = azurerm_container_registry.qwelli.name
}

# Container Instance
output "container_group_name" {
  description = "Container group name"
  value       = azurerm_container_group.qwelli.name
}

output "container_ip_address" {
  description = "Container instance IP address"
  value       = azurerm_container_group.qwelli.ip_address
}

output "container_fqdn" {
  description = "Container instance FQDN (if public access enabled)"
  value       = azurerm_container_group.qwelli.fqdn
}

output "application_url" {
  description = "Application URL"
  value = var.enable_public_access ? (
    azurerm_container_group.qwelli.fqdn != null ?
    "http://${azurerm_container_group.qwelli.fqdn}:8080" :
    "http://${azurerm_container_group.qwelli.ip_address}:8080"
  ) : "Private access only - use VPN or ExpressRoute"
}

# Key Vault
output "key_vault_name" {
  description = "Key Vault name"
  value       = azurerm_key_vault.qwelli.name
}

output "key_vault_uri" {
  description = "Key Vault URI"
  value       = azurerm_key_vault.qwelli.vault_uri
}

# Managed Identity
output "managed_identity_id" {
  description = "User-assigned managed identity ID"
  value       = azurerm_user_assigned_identity.qwelli.id
}

output "managed_identity_principal_id" {
  description = "User-assigned managed identity principal ID"
  value       = azurerm_user_assigned_identity.qwelli.principal_id
}

output "managed_identity_client_id" {
  description = "User-assigned managed identity client ID"
  value       = azurerm_user_assigned_identity.qwelli.client_id
}

# Deployment Instructions
output "deployment_instructions" {
  description = "Next steps for deployment"
  value = <<-EOT

  ═══════════════════════════════════════════════════════════════
                    QWELLI DEPLOYMENT COMPLETE
  ═══════════════════════════════════════════════════════════════

  Resource Group: ${azurerm_resource_group.qwelli.name}
  Location: ${azurerm_resource_group.qwelli.location}

  📦 NEXT STEPS:

  1. Build and push Docker image to ACR:

     az acr login --name ${azurerm_container_registry.qwelli.name}

     docker build -t ${azurerm_container_registry.qwelli.login_server}/${var.container_image_name}:${var.container_image_tag} .

     docker push ${azurerm_container_registry.qwelli.login_server}/${var.container_image_name}:${var.container_image_tag}

  2. Initialize the database:

     psql "postgresql://${var.db_admin_username}:<PASSWORD>@${azurerm_postgresql_flexible_server.qwelli.fqdn}:5432/${var.db_name}?sslmode=require" -f scripts/init-db.sql

  3. Restart container instance to pull new image:

     az container restart --resource-group ${azurerm_resource_group.qwelli.name} --name ${azurerm_container_group.qwelli.name}

  4. Access your application:
     ${var.enable_public_access ? (
       azurerm_container_group.qwelli.fqdn != null ?
       "http://${azurerm_container_group.qwelli.fqdn}:8080" :
       "http://${azurerm_container_group.qwelli.ip_address}:8080"
     ) : "Private access only - configure VPN or ExpressRoute"}

  5. Monitor application:

     # View logs
     az container logs --resource-group ${azurerm_resource_group.qwelli.name} --name ${azurerm_container_group.qwelli.name} --follow

     # Check health
     curl http://${azurerm_container_group.qwelli.fqdn != null ? azurerm_container_group.qwelli.fqdn : azurerm_container_group.qwelli.ip_address}:8080/health

  📊 RESOURCE DETAILS:

  Database:        ${azurerm_postgresql_flexible_server.qwelli.fqdn}
  Container:       ${azurerm_container_group.qwelli.name}
  ACR:             ${azurerm_container_registry.qwelli.login_server}
  Key Vault:       ${azurerm_key_vault.qwelli.name}

  ═══════════════════════════════════════════════════════════════
  EOT
}
