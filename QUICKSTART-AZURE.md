# Qwelli Azure Deployment - Quick Start Guide

This guide will get you up and running with Qwelli on Azure in ~15 minutes.

## Prerequisites

- Azure subscription
- Azure CLI installed (`az`)
- Terraform >= 1.0
- Docker
- Voyage AI API key

## Step-by-Step Deployment

### 1. Login to Azure (1 min)

```bash
az login
az account set --subscription "<your-subscription-id>"
```

### 2. Configure Terraform (2 min)

```bash
cd terraform

# Copy example configuration
cp terraform.tfvars.example terraform.tfvars

# Edit configuration - REQUIRED CHANGES:
# - db_password: Set a secure password (min 12 chars)
# - voyage_api_key: Your Voyage AI API key
nano terraform.tfvars
```

### 3. Deploy Infrastructure (5-10 min)

```bash
# Initialize Terraform
terraform init

# Preview what will be created
terraform plan

# Deploy (takes ~8 minutes)
terraform apply -auto-approve
```

This creates:
✅ Virtual Network with security groups
✅ PostgreSQL database with pgvector
✅ Azure Container Registry
✅ Key Vault for secrets
✅ Container Instance (waiting for app image)

### 4. Build and Deploy Application (3-5 min)

```bash
# Get outputs
ACR_NAME=$(terraform output -raw acr_name)
ACR_SERVER=$(terraform output -raw acr_login_server)
RG_NAME=$(terraform output -raw resource_group_name)
CONTAINER_NAME=$(terraform output -raw container_group_name)

# Login to container registry
az acr login --name $ACR_NAME

# Build and push Docker image
cd ..
docker build -t ${ACR_SERVER}/qwelli:latest .
docker push ${ACR_SERVER}/qwelli:latest

# Restart container to pull image
az container restart --resource-group $RG_NAME --name $CONTAINER_NAME
```

### 5. Initialize Database (1 min)

```bash
# Get database details
cd terraform
DB_HOST=$(terraform output -raw postgres_fqdn)
DB_NAME=$(terraform output -raw postgres_database_name)

# Replace <PASSWORD> with your db_password from terraform.tfvars
psql "postgresql://qwelliadmin:<PASSWORD>@${DB_HOST}:5432/${DB_NAME}?sslmode=require" \
  -f ../scripts/init-db.sql
```

### 6. Verify Deployment (30 sec)

```bash
# Get application URL
terraform output application_url

# Test health endpoint
APP_URL=$(terraform output -raw application_url)
curl ${APP_URL}/health
curl ${APP_URL}/ready
```

You should see:
```json
{"status":"healthy","timestamp":"2024-01-19T..."}
{"status":"ready","timestamp":"2024-01-19T...","config":{...}}
```

### 7. Access Application

Open the URL from step 6 in your browser:
```bash
echo $(terraform output -raw application_url)
```

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                      Azure VNet (10.0.0.0/16)            │
│                                                          │
│  ┌─────────────────┐         ┌──────────────────────┐  │
│  │  Container       │────────▶│  PostgreSQL +        │  │
│  │  Instance        │         │  pgvector            │  │
│  │  (App)           │         │  (Private Endpoint)  │  │
│  └─────────────────┘         └──────────────────────┘  │
│         │                                               │
│         │                    ┌──────────────────────┐  │
│         └───────────────────▶│  Key Vault           │  │
│                               │  (Secrets)           │  │
│                               └──────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                    │
                    ▼
          ┌──────────────────┐
          │  Container        │
          │  Registry (ACR)   │
          └──────────────────┘
```

## Security Features

✅ **Network Isolation**: VNet with private subnets
✅ **Encrypted Secrets**: Azure Key Vault storage
✅ **Managed Identity**: Password-less authentication
✅ **SSL/TLS**: Enforced for all connections
✅ **Private Endpoints**: Database not exposed to internet
✅ **NSG Rules**: Firewall protection

## Common Commands

### View Logs
```bash
az container logs --resource-group $RG_NAME --name $CONTAINER_NAME --follow
```

### Restart Application
```bash
az container restart --resource-group $RG_NAME --name $CONTAINER_NAME
```

### Update Application
```bash
# Build new image
docker build -t ${ACR_SERVER}/qwelli:v2 .
docker push ${ACR_SERVER}/qwelli:v2

# Update terraform vars: container_image_tag = "v2"
# Then apply
terraform apply
```

### Database Connection
```bash
DB_HOST=$(terraform output -raw postgres_fqdn)
psql "postgresql://qwelliadmin:<PASSWORD>@${DB_HOST}:5432/qwelli?sslmode=require"
```

### Check Resource Costs
```bash
az consumption usage list \
  --start-date $(date -d '30 days ago' +%Y-%m-%d) \
  --end-date $(date +%Y-%m-%d) \
  --query "[?contains(instanceName,'qwelli')]" \
  --output table
```

## Troubleshooting

### Container Won't Start
```bash
# Check logs for errors
az container logs --resource-group $RG_NAME --name $CONTAINER_NAME

# Verify secrets are set
az keyvault secret list --vault-name $(terraform output -raw key_vault_name)
```

### Can't Connect to Database
```bash
# Test from your machine
psql "host=$DB_HOST port=5432 dbname=qwelli user=qwelliadmin sslmode=require"

# Check firewall rules
az postgres flexible-server firewall-rule list \
  --resource-group $RG_NAME \
  --name $(terraform output -raw postgres_server_name)
```

### Application Returns 503
```bash
# Check readiness probe
curl ${APP_URL}/ready

# If database is unavailable, check connection string
# and verify pgvector extension is enabled
```

## Scaling

### Vertical Scaling (More Resources)

Edit `terraform.tfvars`:
```hcl
container_cpu    = 4   # Increase from 2
container_memory = 8   # Increase from 4
db_sku_name      = "GP_Standard_D4s_v3"  # Upgrade database
```

Then:
```bash
terraform apply
```

### Database Performance Tuning

```sql
-- Connect to database
psql "postgresql://qwelliadmin:<PASSWORD>@${DB_HOST}:5432/qwelli?sslmode=require"

-- Check HNSW index
SELECT schemaname, tablename, indexname
FROM pg_indexes
WHERE tablename = 'embeddings';

-- Monitor query performance
SELECT * FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;
```

## Production Deployment

For production, use the production variables:

```bash
# Set sensitive vars as environment variables
export TF_VAR_db_password="your-secure-password"
export TF_VAR_voyage_api_key="your-api-key"

# Deploy with production config
terraform apply -var-file="environments/production.tfvars"
```

Production features:
- Private access only (VPN required)
- High availability database
- Premium Container Registry with private endpoint
- Geo-redundant backups
- Higher resource limits

## Cleanup

To remove all resources:

```bash
cd terraform
terraform destroy -auto-approve
```

⚠️ **WARNING**: This deletes everything including the database!

## Next Steps

1. **Set up CI/CD**: Automate deployments with GitHub Actions
2. **Add monitoring**: Configure Azure Monitor and Application Insights
3. **Custom domain**: Map a custom domain name
4. **Backup strategy**: Configure regular database exports
5. **Cost optimization**: Review and optimize resource sizing

## Cost Estimate

**Development** (~$50-100/month):
- PostgreSQL B1ms: ~$15
- Container Instance (1 core, 2GB): ~$20
- ACR Basic: ~$5
- Networking: ~$5-10

**Production** (~$300-500/month):
- PostgreSQL D2s_v3: ~$200
- Container Instance (4 cores, 8GB): ~$150
- ACR Premium: ~$40
- Networking: ~$20-50

## Support

- **Documentation**: See [terraform/README.md](terraform/README.md)
- **Issues**: Check container and database logs
- **Terraform Docs**: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs

---

**That's it!** You now have Qwelli running securely on Azure with:
- ✅ Fully managed PostgreSQL with pgvector
- ✅ Containerized application
- ✅ Automated backups
- ✅ Network security
- ✅ Secret management
- ✅ Health monitoring

Enjoy using Qwelli! 🚀
