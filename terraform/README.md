# Qwelli Terraform Infrastructure

This directory contains Terraform configuration for deploying Qwelli to Azure with full security best practices.

## Architecture

The infrastructure includes:

### Network Layer
- **Virtual Network (VNet)** with subnets for containers and databases
- **Network Security Groups (NSGs)** for traffic control
- **Private DNS Zones** for internal name resolution
- **Private Endpoints** for secure database access (production)

### Security Layer
- **Azure Key Vault** for secret management
- **Managed Identity** for password-less authentication
- **Network isolation** with VNet integration
- **SSL/TLS** encryption for all connections
- **Firewall rules** for database access control

### Data Layer
- **Azure Database for PostgreSQL Flexible Server** with pgvector extension
- **Automated backups** with configurable retention
- **High availability** (zone-redundant in production)
- **Private endpoint** for secure access

### Application Layer
- **Azure Container Instances** with VNet integration
- **Azure Container Registry** for Docker images
- **Health probes** for liveness and readiness
- **Auto-restart** on failure

## Prerequisites

1. **Azure CLI** installed and configured:
   ```bash
   az login
   az account set --subscription "<your-subscription-id>"
   ```

2. **Terraform** >= 1.0 installed:
   ```bash
   # macOS
   brew install terraform

   # Linux
   wget https://releases.hashicorp.com/terraform/1.6.0/terraform_1.6.0_linux_amd64.zip
   unzip terraform_1.6.0_linux_amd64.zip
   sudo mv terraform /usr/local/bin/
   ```

3. **Docker** for building images

4. **Voyage AI API Key** from https://www.voyageai.com/

## Quick Start

### 1. Configure Variables

```bash
# Copy example variables file
cp terraform.tfvars.example terraform.tfvars

# Edit with your values
nano terraform.tfvars
```

Required variables:
- `db_password`: Secure database password (min 12 characters)
- `voyage_api_key`: Your Voyage AI API key
- `allowed_ip_range`: Your IP range for access (use `0.0.0.0/0` for testing, specific IP for production)

### 2. Initialize Terraform

```bash
cd terraform
terraform init
```

### 3. Plan Deployment

```bash
# Review what will be created
terraform plan

# Save plan to file for review
terraform plan -out=tfplan
```

### 4. Deploy Infrastructure

```bash
# Apply the plan
terraform apply

# Or apply with saved plan
terraform apply tfplan
```

This will create:
- Resource group
- Virtual network and subnets
- PostgreSQL database with pgvector
- Container registry
- Key Vault with secrets
- Container instance (waiting for image)

### 5. Build and Push Docker Image

```bash
# Get ACR login server from output
ACR_LOGIN_SERVER=$(terraform output -raw acr_login_server)

# Login to ACR
az acr login --name $(terraform output -raw acr_name)

# Build image
cd ..
docker build -t ${ACR_LOGIN_SERVER}/qwelli:latest .

# Push image
docker push ${ACR_LOGIN_SERVER}/qwelli:latest
```

### 6. Initialize Database

```bash
# Get database connection details
DB_HOST=$(terraform output -raw postgres_fqdn)
DB_NAME=$(terraform output -raw postgres_database_name)
DB_USER="qwelliadmin"  # or your configured username
DB_PASSWORD="<your-password>"

# Connect and run initialization script
psql "postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:5432/${DB_NAME}?sslmode=require" -f ../scripts/init-db.sql
```

### 7. Restart Container

```bash
# Restart container to pull the new image
az container restart \
  --resource-group $(terraform output -raw resource_group_name) \
  --name $(terraform output -raw container_group_name)
```

### 8. Access Application

```bash
# Get application URL
terraform output application_url

# Test health endpoint
curl $(terraform output -raw application_url | sed 's|Private.*|http://$(terraform output -raw container_ip_address):8080|')/health
```

## Environment-Specific Deployments

### Development

```bash
terraform apply -var-file="terraform.tfvars"
```

### Production

```bash
# Set sensitive variables via environment
export TF_VAR_db_password="your-secure-password"
export TF_VAR_voyage_api_key="your-api-key"

# Apply with production config
terraform apply -var-file="environments/production.tfvars"
```

## Security Features

### Network Security

1. **VNet Integration**: All resources deployed in isolated virtual network
2. **Private Endpoints**: Database accessible only within VNet (production)
3. **NSGs**: Firewall rules controlling inbound/outbound traffic
4. **SSL/TLS**: Encrypted connections enforced for database

### Identity and Access

1. **Managed Identity**: Container uses managed identity for ACR and Key Vault
2. **Key Vault**: Secrets stored in Azure Key Vault, not in code
3. **RBAC**: Role-based access control for all resources
4. **No Admin Access**: ACR admin account disabled

### Data Protection

1. **Automated Backups**: Daily backups with 7-35 day retention
2. **Geo-Redundancy**: Enabled in production
3. **Encryption at Rest**: Automatic for all data
4. **Encryption in Transit**: SSL/TLS enforced

## Resource Configuration

### SKU Sizing Guide

#### Development/Testing
```hcl
db_sku_name      = "B_Standard_B1ms"   # 1 vCore, 2 GB RAM
acr_sku          = "Basic"
container_cpu    = 1
container_memory = 2
```

#### Production
```hcl
db_sku_name      = "GP_Standard_D2s_v3"  # 2 vCores, 8 GB RAM
acr_sku          = "Premium"              # For private endpoints
container_cpu    = 4
container_memory = 8
```

#### High-Load Production
```hcl
db_sku_name      = "MO_Standard_E4s_v3"  # 4 vCores, 32 GB RAM
container_cpu    = 4
container_memory = 16
```

## Cost Estimation

### Development (~$50-100/month)
- PostgreSQL (B1ms): ~$15/month
- Container Instance (1 core, 2GB): ~$20/month
- ACR Basic: ~$5/month
- Networking: ~$5-10/month

### Production (~$300-500/month)
- PostgreSQL (D2s_v3): ~$200/month
- Container Instance (4 cores, 8GB): ~$150/month
- ACR Premium: ~$40/month
- Networking + Data Transfer: ~$20-50/month

Use Azure Cost Calculator for detailed estimates: https://azure.microsoft.com/pricing/calculator/

## Monitoring

### View Logs

```bash
# Container logs
az container logs \
  --resource-group $(terraform output -raw resource_group_name) \
  --name $(terraform output -raw container_group_name) \
  --follow

# Database metrics
az monitor metrics list \
  --resource $(terraform output -raw postgres_server_name) \
  --metric-names "cpu_percent,memory_percent,active_connections"
```

### Health Checks

```bash
APP_URL=$(terraform output -raw application_url)

# Liveness
curl ${APP_URL}/health

# Readiness
curl ${APP_URL}/ready
```

## Troubleshooting

### Container Won't Start

```bash
# Check container logs
az container logs --resource-group <rg-name> --name <container-name>

# Verify environment variables
az container show --resource-group <rg-name> --name <container-name> \
  --query "containers[0].environmentVariables"
```

### Database Connection Issues

```bash
# Test database connectivity
psql "host=<db-host> port=5432 dbname=<db-name> user=<username> sslmode=require"

# Check if pgvector is enabled
psql ... -c "SELECT * FROM pg_extension WHERE extname = 'vector';"

# Verify NSG rules allow traffic
az network nsg rule list --resource-group <rg-name> --nsg-name <nsg-name>
```

### Key Vault Access Issues

```bash
# Check managed identity has access
az keyvault show --name <kv-name> --query "properties.accessPolicies"

# Verify secrets exist
az keyvault secret list --vault-name <kv-name>
```

## Maintenance

### Update Container Image

```bash
# Build new image
docker build -t <acr-server>/qwelli:v2 .
docker push <acr-server>/qwelli:v2

# Update Terraform variables
# container_image_tag = "v2"

# Apply changes
terraform apply
```

### Database Backup

```bash
# Create manual backup
az postgres flexible-server backup create \
  --resource-group <rg-name> \
  --name <server-name> \
  --backup-name "manual-$(date +%Y%m%d)"
```

### Scale Resources

```bash
# Update terraform.tfvars with new sizes
container_cpu    = 4
container_memory = 8

# Apply changes
terraform apply
```

## Cleanup

### Destroy All Resources

```bash
# WARNING: This will delete everything!
terraform destroy

# Or with specific var file
terraform destroy -var-file="environments/production.tfvars"
```

### Selective Cleanup

```bash
# Remove specific resource
terraform destroy -target=azurerm_container_group.qwelli

# Keep data, remove compute
terraform destroy \
  -target=azurerm_container_group.qwelli \
  -target=azurerm_container_registry.qwelli
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Deploy to Azure

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Azure Login
        uses: azure/login@v1
        with:
          creds: ${{ secrets.AZURE_CREDENTIALS }}

      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v2

      - name: Terraform Init
        run: terraform init
        working-directory: ./terraform

      - name: Terraform Apply
        run: terraform apply -auto-approve
        working-directory: ./terraform
        env:
          TF_VAR_db_password: ${{ secrets.DB_PASSWORD }}
          TF_VAR_voyage_api_key: ${{ secrets.VOYAGE_API_KEY }}
```

## Best Practices

1. **Never commit** `terraform.tfvars` or `.tfstate` files
2. **Use remote state** for team collaboration
3. **Enable soft delete** on Key Vault for production
4. **Use variables** for environment-specific configuration
5. **Tag all resources** for cost tracking
6. **Review plans** before applying
7. **Use workspaces** for multi-environment deployments
8. **Enable Azure Monitor** for production workloads

## Advanced Configuration

### Remote State Backend

Create `backend.tf`:

```hcl
terraform {
  backend "azurerm" {
    resource_group_name  = "terraform-state-rg"
    storage_account_name = "tfstate<unique-id>"
    container_name       = "tfstate"
    key                  = "qwelli.tfstate"
  }
}
```

### Multi-Region Deployment

Use Terraform workspaces:

```bash
# Create workspaces
terraform workspace new eastus
terraform workspace new westus

# Deploy to specific region
terraform workspace select eastus
terraform apply -var="location=eastus"
```

## Support

For issues or questions:
- Check the [main README](../README.md)
- Review Terraform [output messages](#outputs)
- Check Azure Portal for resource status
- Review container and database logs

## License

See main project LICENSE file.
