# AKS Cluster Outputs
output "aks_cluster_name" {
  description = "AKS cluster name"
  value       = azurerm_kubernetes_cluster.qwelli.name
}

output "aks_cluster_id" {
  description = "AKS cluster ID"
  value       = azurerm_kubernetes_cluster.qwelli.id
}

output "aks_fqdn" {
  description = "AKS cluster FQDN"
  value       = azurerm_kubernetes_cluster.qwelli.fqdn
}

output "aks_node_resource_group" {
  description = "AKS node resource group"
  value       = azurerm_kubernetes_cluster.qwelli.node_resource_group
}

output "kube_config" {
  description = "Kubernetes config for kubectl"
  value       = azurerm_kubernetes_cluster.qwelli.kube_config_raw
  sensitive   = true
}

output "get_credentials_command" {
  description = "Command to get AKS credentials"
  value       = "az aks get-credentials --resource-group ${azurerm_resource_group.qwelli.name} --name ${azurerm_kubernetes_cluster.qwelli.name}"
}

# Database Outputs
output "postgres_fqdn" {
  description = "PostgreSQL server FQDN"
  value       = azurerm_postgresql_flexible_server.qwelli.fqdn
}

output "postgres_connection_string" {
  description = "PostgreSQL connection string"
  value       = "postgresql://${var.db_admin_username}@${azurerm_postgresql_flexible_server.qwelli.fqdn}:5432/${var.db_name}?sslmode=require"
  sensitive   = true
}

# ACR Outputs
output "acr_login_server" {
  description = "ACR login server"
  value       = azurerm_container_registry.qwelli.login_server
}

output "acr_name" {
  description = "ACR name"
  value       = azurerm_container_registry.qwelli.name
}

# Key Vault Outputs
output "key_vault_name" {
  description = "Key Vault name"
  value       = azurerm_key_vault.qwelli.name
}

output "key_vault_uri" {
  description = "Key Vault URI"
  value       = azurerm_key_vault.qwelli.vault_uri
}

# Workload Identity Outputs
output "pod_identity_client_id" {
  description = "Pod identity client ID for workload identity"
  value       = azurerm_user_assigned_identity.qwelli_pod.client_id
}

output "pod_identity_tenant_id" {
  description = "Pod identity tenant ID"
  value       = data.azurerm_client_config.current.tenant_id
}

# Network Outputs
output "vnet_id" {
  description = "Virtual Network ID"
  value       = azurerm_virtual_network.qwelli.id
}

output "vnet_name" {
  description = "Virtual Network name"
  value       = azurerm_virtual_network.qwelli.name
}

# Monitoring Outputs
output "log_analytics_workspace_id" {
  description = "Log Analytics Workspace ID"
  value       = azurerm_log_analytics_workspace.qwelli.id
}

# Deployment Instructions
output "deployment_instructions" {
  description = "Next steps for deployment"
  value = <<-EOT

  ═══════════════════════════════════════════════════════════════
                QWELLI AKS DEPLOYMENT COMPLETE
  ═══════════════════════════════════════════════════════════════

  Resource Group: ${azurerm_resource_group.qwelli.name}
  Location: ${azurerm_resource_group.qwelli.location}
  AKS Cluster: ${azurerm_kubernetes_cluster.qwelli.name}

  📦 NEXT STEPS:

  1. Get AKS credentials:

     az aks get-credentials \
       --resource-group ${azurerm_resource_group.qwelli.name} \
       --name ${azurerm_kubernetes_cluster.qwelli.name}

  2. Verify cluster access:

     kubectl get nodes
     kubectl get namespaces

  3. Build and push Docker image:

     az acr login --name ${azurerm_container_registry.qwelli.name}

     docker build -t ${azurerm_container_registry.qwelli.login_server}/qwelli:latest .

     docker push ${azurerm_container_registry.qwelli.login_server}/qwelli:latest

  4. Initialize database:

     psql "postgresql://${var.db_admin_username}:<PASSWORD>@${azurerm_postgresql_flexible_server.qwelli.fqdn}:5432/${var.db_name}?sslmode=require" \
       -f ../../scripts/init-db.sql

  5. Deploy to Kubernetes:

     # Update k8s/deployment.yaml with your values, then:
     kubectl apply -f ../../k8s/

  6. Get service URL:

     kubectl get service qwelli-service -n ${var.kubernetes_namespace}

  7. Monitor pods:

     kubectl get pods -n ${var.kubernetes_namespace} -w
     kubectl logs -f deployment/qwelli -n ${var.kubernetes_namespace}

  📊 SCALING CONFIGURATION:

  MVP (Minimal Cost ~$70/month):
  - System nodes: ${var.system_node_min_count}-${var.system_node_max_count} x ${var.system_node_size}
  - User nodes: ${var.user_node_min_count}-${var.user_node_max_count} x ${var.user_node_size}
  - Database: ${var.db_sku_name}

  To scale up for production:
  - Update terraform.tfvars with larger VM sizes
  - Increase node pool max counts
  - Upgrade database SKU to GP_Standard_D2s_v3 or higher
  - Enable high availability (zone-redundant)

  🔒 SECURITY FEATURES ENABLED:

  ✅ Private VNet integration
  ✅ Azure CNI networking
  ✅ Network policies for pod-to-pod security
  ✅ Workload identity for Key Vault access
  ✅ Private database endpoint
  ✅ Azure AD RBAC
  ✅ Container insights monitoring
  ✅ Network Security Groups

  📈 MONITORING:

  View logs and metrics:
  - Azure Portal: Container Insights
  - CLI: az monitor log-analytics query

  Workspace ID: ${azurerm_log_analytics_workspace.qwelli.workspace_id}

  ═══════════════════════════════════════════════════════════════
  EOT
}
