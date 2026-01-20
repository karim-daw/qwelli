# General Configuration
variable "resource_group_name" {
  description = "Name of the resource group"
  type        = string
  default     = "qwelli-rg"
}

variable "location" {
  description = "Azure region for resources"
  type        = string
  default     = "eastus"
}

variable "prefix" {
  description = "Prefix for resource names"
  type        = string
  default     = "qwelli"

  validation {
    condition     = length(var.prefix) <= 10 && can(regex("^[a-z0-9]+$", var.prefix))
    error_message = "Prefix must be lowercase alphanumeric and max 10 characters"
  }
}

variable "environment" {
  description = "Environment name (development, staging, production)"
  type        = string
  default     = "development"

  validation {
    condition     = contains(["development", "staging", "production"], var.environment)
    error_message = "Environment must be development, staging, or production"
  }
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default = {
    Environment = "development"
    Application = "Qwelli"
    ManagedBy   = "Terraform"
  }
}

# Network Configuration
variable "allowed_ip_range" {
  description = "Allowed IP range for access (CIDR notation)"
  type        = string
  default     = "*"  # Change this to restrict access
}

variable "enable_public_access" {
  description = "Enable public IP for container instance"
  type        = bool
  default     = true  # Set to false for production with VPN/ExpressRoute
}

# Database Configuration
variable "db_name" {
  description = "PostgreSQL database name"
  type        = string
  default     = "qwelli"
}

variable "db_admin_username" {
  description = "PostgreSQL admin username"
  type        = string
  default     = "qwelliadmin"
  sensitive   = true
}

variable "db_password" {
  description = "PostgreSQL admin password"
  type        = string
  sensitive   = true

  validation {
    condition     = length(var.db_password) >= 12
    error_message = "Database password must be at least 12 characters long"
  }
}

variable "db_sku_name" {
  description = "PostgreSQL SKU name (tier_family_cores)"
  type        = string
  default     = "B_Standard_B1ms"  # Burstable tier for dev/test

  # Common SKUs:
  # - B_Standard_B1ms: Burstable, 1 vCore, 2 GiB RAM
  # - GP_Standard_D2s_v3: General Purpose, 2 vCores, 8 GiB RAM
  # - MO_Standard_E4s_v3: Memory Optimized, 4 vCores, 32 GiB RAM
}

variable "db_storage_mb" {
  description = "PostgreSQL storage in MB"
  type        = number
  default     = 32768  # 32 GB

  validation {
    condition     = var.db_storage_mb >= 32768 && var.db_storage_mb <= 16777216
    error_message = "Storage must be between 32 GB and 16 TB"
  }
}

variable "db_backup_retention_days" {
  description = "Number of days to retain backups"
  type        = number
  default     = 7

  validation {
    condition     = var.db_backup_retention_days >= 7 && var.db_backup_retention_days <= 35
    error_message = "Backup retention must be between 7 and 35 days"
  }
}

variable "db_max_connections" {
  description = "Maximum number of database connections"
  type        = string
  default     = "100"
}

variable "allow_azure_services" {
  description = "Allow Azure services to access database"
  type        = bool
  default     = true
}

# Container Registry Configuration
variable "acr_sku" {
  description = "Azure Container Registry SKU"
  type        = string
  default     = "Basic"  # Basic, Standard, or Premium

  validation {
    condition     = contains(["Basic", "Standard", "Premium"], var.acr_sku)
    error_message = "ACR SKU must be Basic, Standard, or Premium"
  }
}

# Container Configuration
variable "container_image_name" {
  description = "Container image name"
  type        = string
  default     = "qwelli"
}

variable "container_image_tag" {
  description = "Container image tag"
  type        = string
  default     = "latest"
}

variable "container_cpu" {
  description = "Number of CPU cores for container"
  type        = number
  default     = 2

  validation {
    condition     = var.container_cpu >= 1 && var.container_cpu <= 4
    error_message = "Container CPU must be between 1 and 4 cores"
  }
}

variable "container_memory" {
  description = "Memory in GB for container"
  type        = number
  default     = 4

  validation {
    condition     = var.container_memory >= 1 && var.container_memory <= 16
    error_message = "Container memory must be between 1 and 16 GB"
  }
}

# Application Configuration
variable "voyage_api_key" {
  description = "Voyage AI API key"
  type        = string
  sensitive   = true

  validation {
    condition     = length(var.voyage_api_key) > 0
    error_message = "Voyage API key cannot be empty"
  }
}

variable "embedding_model" {
  description = "Embedding model to use"
  type        = string
  default     = "voyage-multimodal-3"
}

variable "enable_multimodal" {
  description = "Enable multimodal embeddings (images)"
  type        = string
  default     = "true"
}

variable "enable_reranker" {
  description = "Enable reranking of search results"
  type        = string
  default     = "true"
}

variable "image_quality" {
  description = "Image quality setting (low, medium, high)"
  type        = string
  default     = "medium"

  validation {
    condition     = contains(["low", "medium", "high"], var.image_quality)
    error_message = "Image quality must be low, medium, or high"
  }
}

# Monitoring and Alerting
variable "alert_email_addresses" {
  description = "Email addresses for alerts"
  type        = list(string)
  default     = []
}
