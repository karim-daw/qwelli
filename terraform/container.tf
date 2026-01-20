# Azure Container Instances for running Qwelli application
resource "azurerm_container_group" "qwelli" {
  name                = "${var.prefix}-aci"
  location            = azurerm_resource_group.qwelli.location
  resource_group_name = azurerm_resource_group.qwelli.name
  os_type             = "Linux"
  restart_policy      = "Always"

  # Network configuration - VNet integration
  subnet_ids = [azurerm_subnet.containers.id]

  # Use managed identity
  identity {
    type = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.qwelli.id]
  }

  # Image registry credentials using managed identity
  image_registry_credential {
    server                    = azurerm_container_registry.qwelli.login_server
    user_assigned_identity_id = azurerm_user_assigned_identity.qwelli.id
  }

  # Qwelli application container
  container {
    name   = "qwelli-app"
    image  = "${azurerm_container_registry.qwelli.login_server}/${var.container_image_name}:${var.container_image_tag}"
    cpu    = var.container_cpu
    memory = var.container_memory

    ports {
      port     = 8080
      protocol = "TCP"
    }

    # Environment variables (non-sensitive)
    environment_variables = {
      DB_TYPE               = "postgres"
      DB_HOST               = azurerm_postgresql_flexible_server.qwelli.fqdn
      DB_PORT               = "5432"
      DB_NAME               = var.db_name
      DB_USER               = var.db_admin_username
      DB_SSLMODE            = "require"
      DB_MAX_CONNS          = var.db_max_connections
      DB_MAX_IDLE           = "5"
      QWELLI_EMBEDDING_MODEL = var.embedding_model
      ENABLE_MULTIMODAL     = var.enable_multimodal
      ENABLE_RERANKER       = var.enable_reranker
      IMAGE_QUALITY         = var.image_quality
      SERVER_HOST           = "0.0.0.0"
      SERVER_PORT           = "8080"
    }

    # Secure environment variables from Key Vault
    secure_environment_variables = {
      DB_PASSWORD           = azurerm_key_vault_secret.db_password.value
      QWELLI_EMBEDDING_KEY = azurerm_key_vault_secret.voyage_api_key.value
    }

    # Liveness probe
    liveness_probe {
      http_get {
        path   = "/health"
        port   = 8080
        scheme = "Http"
      }
      initial_delay_seconds = 30
      period_seconds        = 10
      failure_threshold     = 3
      timeout_seconds       = 3
    }

    # Readiness probe
    readiness_probe {
      http_get {
        path   = "/ready"
        port   = 8080
        scheme = "Http"
      }
      initial_delay_seconds = 5
      period_seconds        = 5
      failure_threshold     = 3
      timeout_seconds       = 3
    }
  }

  # DNS name label for public access
  dns_name_label = var.enable_public_access ? "${var.prefix}-${random_string.suffix.result}" : null

  # Expose ports for public access if enabled
  dynamic "ip_address_type" {
    for_each = var.enable_public_access ? ["Public"] : ["Private"]
    content {
      value = ip_address_type.value
    }
  }

  tags = var.tags

  depends_on = [
    azurerm_postgresql_flexible_server.qwelli,
    azurerm_postgresql_flexible_server_database.qwelli,
    azurerm_postgresql_flexible_server_configuration.azure_extensions,
    azurerm_key_vault_access_policy.container,
    azurerm_role_assignment.acr_pull
  ]
}
