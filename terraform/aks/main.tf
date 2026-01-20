terraform {
  required_version = ">= 1.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}

provider "azurerm" {
  features {
    key_vault {
      purge_soft_delete_on_destroy = true
    }
  }
}

# Configure Kubernetes provider after AKS cluster is created
provider "kubernetes" {
  host                   = azurerm_kubernetes_cluster.qwelli.kube_config[0].host
  client_certificate     = base64decode(azurerm_kubernetes_cluster.qwelli.kube_config[0].client_certificate)
  client_key             = base64decode(azurerm_kubernetes_cluster.qwelli.kube_config[0].client_key)
  cluster_ca_certificate = base64decode(azurerm_kubernetes_cluster.qwelli.kube_config[0].cluster_ca_certificate)
}

provider "helm" {
  kubernetes {
    host                   = azurerm_kubernetes_cluster.qwelli.kube_config[0].host
    client_certificate     = base64decode(azurerm_kubernetes_cluster.qwelli.kube_config[0].client_certificate)
    client_key             = base64decode(azurerm_kubernetes_cluster.qwelli.kube_config[0].client_key)
    cluster_ca_certificate = base64decode(azurerm_kubernetes_cluster.qwelli.kube_config[0].cluster_ca_certificate)
  }
}

data "azurerm_client_config" "current" {}

resource "random_string" "suffix" {
  length  = 8
  special = false
  upper   = false
}

# Resource Group
resource "azurerm_resource_group" "qwelli" {
  name     = var.resource_group_name
  location = var.location

  tags = merge(
    var.tags,
    {
      ManagedBy = "Terraform"
      Platform  = "AKS"
    }
  )
}

# Virtual Network for AKS
resource "azurerm_virtual_network" "qwelli" {
  name                = "${var.prefix}-vnet"
  location            = azurerm_resource_group.qwelli.location
  resource_group_name = azurerm_resource_group.qwelli.name
  address_space       = ["10.0.0.0/8"]

  tags = var.tags
}

# Subnet for AKS nodes
resource "azurerm_subnet" "aks_nodes" {
  name                 = "aks-nodes-subnet"
  resource_group_name  = azurerm_resource_group.qwelli.name
  virtual_network_name = azurerm_virtual_network.qwelli.name
  address_prefixes     = ["10.240.0.0/16"]

  # Service endpoints for secure access to Azure services
  service_endpoints = [
    "Microsoft.ContainerRegistry",
    "Microsoft.KeyVault",
    "Microsoft.Storage"
  ]
}

# Subnet for PostgreSQL
resource "azurerm_subnet" "database" {
  name                 = "database-subnet"
  resource_group_name  = azurerm_resource_group.qwelli.name
  virtual_network_name = azurerm_virtual_network.qwelli.name
  address_prefixes     = ["10.241.0.0/24"]

  delegation {
    name = "postgres-delegation"
    service_delegation {
      name = "Microsoft.DBforPostgreSQL/flexibleServers"
      actions = [
        "Microsoft.Network/virtualNetworks/subnets/join/action"
      ]
    }
  }
}

# Subnet for Application Gateway (optional - for ingress)
resource "azurerm_subnet" "appgw" {
  count                = var.enable_application_gateway ? 1 : 0
  name                 = "appgw-subnet"
  resource_group_name  = azurerm_resource_group.qwelli.name
  virtual_network_name = azurerm_virtual_network.qwelli.name
  address_prefixes     = ["10.242.0.0/24"]
}

# Private DNS Zone for PostgreSQL
resource "azurerm_private_dns_zone" "postgres" {
  name                = "privatelink.postgres.database.azure.com"
  resource_group_name = azurerm_resource_group.qwelli.name

  tags = var.tags
}

resource "azurerm_private_dns_zone_virtual_network_link" "postgres" {
  name                  = "${var.prefix}-postgres-dns-link"
  resource_group_name   = azurerm_resource_group.qwelli.name
  private_dns_zone_name = azurerm_private_dns_zone.postgres.name
  virtual_network_id    = azurerm_virtual_network.qwelli.id

  tags = var.tags
}

# Network Security Group for AKS subnet
resource "azurerm_network_security_group" "aks" {
  name                = "${var.prefix}-aks-nsg"
  location            = azurerm_resource_group.qwelli.location
  resource_group_name = azurerm_resource_group.qwelli.name

  # Allow internal traffic
  security_rule {
    name                       = "AllowVnetInbound"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefix      = "VirtualNetwork"
    destination_address_prefix = "VirtualNetwork"
  }

  # Allow Azure Load Balancer
  security_rule {
    name                       = "AllowAzureLoadBalancer"
    priority                   = 110
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefix      = "AzureLoadBalancer"
    destination_address_prefix = "*"
  }

  # Deny all other inbound (if using private cluster)
  security_rule {
    name                       = "DenyAllInbound"
    priority                   = 4096
    direction                  = "Inbound"
    access                     = "Deny"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  tags = var.tags
}

resource "azurerm_subnet_network_security_group_association" "aks" {
  subnet_id                 = azurerm_subnet.aks_nodes.id
  network_security_group_id = azurerm_network_security_group.aks.id
}
