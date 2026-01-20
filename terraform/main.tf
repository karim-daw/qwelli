terraform {
  required_version = ">= 1.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }

  # Optional: Configure remote backend for state storage
  # backend "azurerm" {
  #   resource_group_name  = "terraform-state-rg"
  #   storage_account_name = "tfstate"
  #   container_name       = "tfstate"
  #   key                  = "qwelli.tfstate"
  # }
}

provider "azurerm" {
  features {
    key_vault {
      purge_soft_delete_on_destroy = true
    }
    resource_group {
      prevent_deletion_if_contains_resources = false
    }
  }
}

# Data source for current client
data "azurerm_client_config" "current" {}

# Random suffix for globally unique names
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
      Project   = "Qwelli"
    }
  )
}

# Virtual Network for secure networking
resource "azurerm_virtual_network" "qwelli" {
  name                = "${var.prefix}-vnet"
  location            = azurerm_resource_group.qwelli.location
  resource_group_name = azurerm_resource_group.qwelli.name
  address_space       = ["10.0.0.0/16"]

  tags = var.tags
}

# Subnet for Container Instances
resource "azurerm_subnet" "containers" {
  name                 = "containers-subnet"
  resource_group_name  = azurerm_resource_group.qwelli.name
  virtual_network_name = azurerm_virtual_network.qwelli.name
  address_prefixes     = ["10.0.1.0/24"]

  delegation {
    name = "container-delegation"
    service_delegation {
      name    = "Microsoft.ContainerInstance/containerGroups"
      actions = ["Microsoft.Network/virtualNetworks/subnets/action"]
    }
  }
}

# Subnet for PostgreSQL Private Endpoint
resource "azurerm_subnet" "database" {
  name                 = "database-subnet"
  resource_group_name  = azurerm_resource_group.qwelli.name
  virtual_network_name = azurerm_virtual_network.qwelli.name
  address_prefixes     = ["10.0.2.0/24"]
}

# Network Security Group for Container Subnet
resource "azurerm_network_security_group" "containers" {
  name                = "${var.prefix}-containers-nsg"
  location            = azurerm_resource_group.qwelli.location
  resource_group_name = azurerm_resource_group.qwelli.name

  security_rule {
    name                       = "AllowHTTP"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "8080"
    source_address_prefix      = var.allowed_ip_range
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "AllowHTTPS"
    priority                   = 110
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "443"
    source_address_prefix      = var.allowed_ip_range
    destination_address_prefix = "*"
  }

  tags = var.tags
}

# Associate NSG with Container Subnet
resource "azurerm_subnet_network_security_group_association" "containers" {
  subnet_id                 = azurerm_subnet.containers.id
  network_security_group_id = azurerm_network_security_group.containers.id
}

# Private DNS Zone for PostgreSQL
resource "azurerm_private_dns_zone" "postgres" {
  name                = "privatelink.postgres.database.azure.com"
  resource_group_name = azurerm_resource_group.qwelli.name

  tags = var.tags
}

# Link Private DNS Zone to VNet
resource "azurerm_private_dns_zone_virtual_network_link" "postgres" {
  name                  = "${var.prefix}-postgres-dns-link"
  resource_group_name   = azurerm_resource_group.qwelli.name
  private_dns_zone_name = azurerm_private_dns_zone.postgres.name
  virtual_network_id    = azurerm_virtual_network.qwelli.id

  tags = var.tags
}
