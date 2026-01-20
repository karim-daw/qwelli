# General Configuration
variable "resource_group_name" {
  description = "Name of the resource group"
  type        = string
  default     = "qwelli-aks-rg"
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
    Platform    = "AKS"
  }
}

# Network Configuration
variable "allowed_ip_range" {
  description = "Allowed IP range for access (CIDR notation)"
  type        = string
  default     = "0.0.0.0/0"
}

# AKS Configuration
variable "kubernetes_version" {
  description = "Kubernetes version"
  type        = string
  default     = "1.28"  # Use latest stable version
}

variable "enable_private_cluster" {
  description = "Enable private AKS cluster (API server not publicly accessible)"
  type        = bool
  default     = false  # Set true for production
}

variable "aks_admin_group_object_ids" {
  description = "List of Azure AD group object IDs for AKS cluster admin access"
  type        = list(string)
  default     = []
}

# System Node Pool (for system workloads)
variable "system_node_size" {
  description = "VM size for system nodes"
  type        = string
  default     = "Standard_D2s_v3"  # 2 vCPU, 8GB RAM - minimum for AKS
}

variable "system_node_count" {
  description = "Initial number of system nodes"
  type        = number
  default     = 1

  validation {
    condition     = var.system_node_count >= 1 && var.system_node_count <= 10
    error_message = "System node count must be between 1 and 10"
  }
}

variable "system_node_min_count" {
  description = "Minimum number of system nodes (autoscaling)"
  type        = number
  default     = 1
}

variable "system_node_max_count" {
  description = "Maximum number of system nodes (autoscaling)"
  type        = number
  default     = 3
}

# User Node Pool (for application workloads)
variable "user_node_size" {
  description = "VM size for user nodes"
  type        = string
  default     = "Standard_D2s_v3"  # 2 vCPU, 8GB RAM

  # Options for scaling:
  # - Standard_B2s: 2 vCPU, 4GB RAM (cheapest, ~$30/month)
  # - Standard_D2s_v3: 2 vCPU, 8GB RAM (~$70/month)
  # - Standard_D4s_v3: 4 vCPU, 16GB RAM (~$140/month)
  # - Standard_D8s_v3: 8 vCPU, 32GB RAM (~$280/month)
}

variable "user_node_count" {
  description = "Initial number of user nodes"
  type        = number
  default     = 1
}

variable "user_node_min_count" {
  description = "Minimum number of user nodes (autoscaling)"
  type        = number
  default     = 1
}

variable "user_node_max_count" {
  description = "Maximum number of user nodes (autoscaling)"
  type        = number
  default     = 5
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
  description = "PostgreSQL SKU name"
  type        = string
  default     = "B_Standard_B1ms"  # Burstable for MVP, upgrade to GP_Standard_D2s_v3 for production
}

variable "db_storage_mb" {
  description = "PostgreSQL storage in MB"
  type        = number
  default     = 32768  # 32 GB
}

variable "db_backup_retention_days" {
  description = "Number of days to retain backups"
  type        = number
  default     = 7
}

variable "db_max_connections" {
  description = "Maximum number of database connections"
  type        = string
  default     = "100"
}

# Container Registry Configuration
variable "acr_sku" {
  description = "Azure Container Registry SKU"
  type        = string
  default     = "Basic"  # Basic for MVP, Premium for production with private endpoint

  validation {
    condition     = contains(["Basic", "Standard", "Premium"], var.acr_sku)
    error_message = "ACR SKU must be Basic, Standard, or Premium"
  }
}

# Application Configuration
variable "voyage_api_key" {
  description = "Voyage AI API key"
  type        = string
  sensitive   = true
}

variable "embedding_model" {
  description = "Embedding model to use"
  type        = string
  default     = "voyage-multimodal-3"
}

variable "enable_multimodal" {
  description = "Enable multimodal embeddings"
  type        = bool
  default     = true
}

variable "enable_reranker" {
  description = "Enable reranking"
  type        = bool
  default     = true
}

variable "image_quality" {
  description = "Image quality setting"
  type        = string
  default     = "medium"
}

# Kubernetes Configuration
variable "kubernetes_namespace" {
  description = "Kubernetes namespace for Qwelli application"
  type        = string
  default     = "qwelli"
}

# Ingress Configuration
variable "enable_application_gateway" {
  description = "Enable Azure Application Gateway as ingress controller"
  type        = bool
  default     = false  # Set true for production with custom domain
}

variable "enable_cert_manager" {
  description = "Enable cert-manager for automatic SSL certificates"
  type        = bool
  default     = false
}
