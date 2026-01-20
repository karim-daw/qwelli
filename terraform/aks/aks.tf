# Azure Kubernetes Service Cluster
resource "azurerm_kubernetes_cluster" "qwelli" {
  name                = "${var.prefix}-aks"
  location            = azurerm_resource_group.qwelli.location
  resource_group_name = azurerm_resource_group.qwelli.name
  dns_prefix          = "${var.prefix}-aks"
  kubernetes_version  = var.kubernetes_version

  # Private cluster configuration for enhanced security
  private_cluster_enabled = var.enable_private_cluster

  # System-assigned managed identity
  identity {
    type = "SystemAssigned"
  }

  # Default node pool (system pool)
  default_node_pool {
    name                = "system"
    node_count          = var.system_node_count
    vm_size             = var.system_node_size
    vnet_subnet_id      = azurerm_subnet.aks_nodes.id
    enable_auto_scaling = true
    min_count           = var.system_node_min_count
    max_count           = var.system_node_max_count
    max_pods            = 110
    os_disk_size_gb     = 128
    os_disk_type        = "Managed"
    type                = "VirtualMachineScaleSets"

    # Node labels for system workloads
    node_labels = {
      "workload-type" = "system"
    }

    # Upgrade settings
    upgrade_settings {
      max_surge = "33%"
    }

    tags = var.tags
  }

  # Network configuration
  network_profile {
    network_plugin    = "azure"  # Azure CNI for better VNet integration
    network_policy    = "azure"  # Network policies for pod-to-pod security
    dns_service_ip    = "10.0.0.10"
    service_cidr      = "10.0.0.0/16"
    load_balancer_sku = "standard"
    outbound_type     = "loadBalancer"

    # Load balancer profile
    load_balancer_profile {
      managed_outbound_ip_count = 1
    }
  }

  # Azure AD integration for RBAC
  azure_active_directory_role_based_access_control {
    managed                = true
    azure_rbac_enabled     = true
    admin_group_object_ids = var.aks_admin_group_object_ids
  }

  # Azure Monitor for container insights
  oms_agent {
    log_analytics_workspace_id = azurerm_log_analytics_workspace.qwelli.id
  }

  # Key Vault secrets provider
  key_vault_secrets_provider {
    secret_rotation_enabled  = true
    secret_rotation_interval = "2m"
  }

  # Workload identity (recommended for pod authentication)
  workload_identity_enabled = true
  oidc_issuer_enabled       = true

  # Auto-upgrade configuration
  automatic_channel_upgrade = var.environment == "production" ? "stable" : "none"

  maintenance_window {
    allowed {
      day   = "Sunday"
      hours = [2, 3, 4]
    }
  }

  # Security features
  azure_policy_enabled = var.environment == "production"

  # HTTP application routing (disabled for production)
  http_application_routing_enabled = false

  # Additional security settings
  local_account_disabled = var.environment == "production"

  tags = var.tags
}

# User node pool for application workloads
resource "azurerm_kubernetes_cluster_node_pool" "user" {
  name                  = "user"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.qwelli.id
  vm_size               = var.user_node_size
  node_count            = var.user_node_count
  vnet_subnet_id        = azurerm_subnet.aks_nodes.id
  enable_auto_scaling   = true
  min_count             = var.user_node_min_count
  max_count             = var.user_node_max_count
  max_pods              = 110
  os_disk_size_gb       = 128
  os_type               = "Linux"
  mode                  = "User"

  # Node labels for application workloads
  node_labels = {
    "workload-type" = "application"
  }

  # Taints to ensure only application pods run here
  node_taints = []

  upgrade_settings {
    max_surge = "33%"
  }

  tags = var.tags
}

# Log Analytics Workspace for monitoring
resource "azurerm_log_analytics_workspace" "qwelli" {
  name                = "${var.prefix}-logs"
  location            = azurerm_resource_group.qwelli.location
  resource_group_name = azurerm_resource_group.qwelli.name
  sku                 = "PerGB2018"
  retention_in_days   = var.environment == "production" ? 90 : 30

  tags = var.tags
}

# Diagnostic settings for AKS
resource "azurerm_monitor_diagnostic_setting" "aks" {
  name                       = "${var.prefix}-aks-diagnostics"
  target_resource_id         = azurerm_kubernetes_cluster.qwelli.id
  log_analytics_workspace_id = azurerm_log_analytics_workspace.qwelli.id

  enabled_log {
    category = "kube-apiserver"
  }

  enabled_log {
    category = "kube-controller-manager"
  }

  enabled_log {
    category = "kube-scheduler"
  }

  enabled_log {
    category = "kube-audit"
  }

  enabled_log {
    category = "cluster-autoscaler"
  }

  metric {
    category = "AllMetrics"
    enabled  = true
  }
}

# Grant AKS access to ACR
resource "azurerm_role_assignment" "aks_acr" {
  principal_id                     = azurerm_kubernetes_cluster.qwelli.kubelet_identity[0].object_id
  role_definition_name             = "AcrPull"
  scope                            = azurerm_container_registry.qwelli.id
  skip_service_principal_aad_check = true
}

# Grant AKS access to VNet for network policies
resource "azurerm_role_assignment" "aks_network" {
  principal_id                     = azurerm_kubernetes_cluster.qwelli.identity[0].principal_id
  role_definition_name             = "Network Contributor"
  scope                            = azurerm_virtual_network.qwelli.id
  skip_service_principal_aad_check = true
}
