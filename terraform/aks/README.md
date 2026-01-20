# Qwelli on Azure Kubernetes Service (AKS)

Complete infrastructure-as-code for deploying Qwelli on AKS with enterprise-grade security and scalability.

## 🎯 Why AKS?

### ✅ Scalability
- **Horizontal Pod Autoscaler (HPA)**: Automatically scale from 2 to 10+ pods based on load
- **Cluster Autoscaler**: Add/remove nodes automatically
- **Scale to zero**: Reduce costs during idle periods
- **Global distribution**: Multi-region deployments

### ✅ Security
- **VNet Integration**: All resources in isolated virtual network
- **Private Cluster**: API server not exposed to internet
- **Network Policies**: Pod-to-pod traffic control
- **Workload Identity**: Password-less authentication with Key Vault
- **Azure CNI**: Advanced networking with network security groups
- **Private Endpoints**: Database accessible only within VNet

### ✅ Cost Efficiency
- **MVP**: Start at ~$70-100/month
- **Production**: Scale to $500-800/month with HA
- **Pay for what you use**: Auto-scaling reduces waste
- **No cluster management fees**: AKS control plane is FREE

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│          Azure Virtual Network (10.0.0.0/8)                     │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │  AKS Cluster (Private)                                     │ │
│  │                                                            │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │ │
│  │  │ Qwelli Pod  │  │ Qwelli Pod  │  │ Qwelli Pod  │       │ │
│  │  │ (App)       │  │ (App)       │  │ (App)       │       │ │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘       │ │
│  │         │                │                │               │ │
│  │         └────────────────┴────────────────┘               │ │
│  │                          │                                │ │
│  │                 ┌────────▼────────┐                       │ │
│  │                 │  Load Balancer  │                       │ │
│  │                 └────────┬────────┘                       │ │
│  │                          │                                │ │
│  └──────────────────────────┼────────────────────────────────┘ │
│                             │                                  │
│         ┌───────────────────┼───────────────────┐              │
│         │                   │                   │              │
│    ┌────▼─────┐      ┌─────▼──────┐     ┌─────▼────┐         │
│    │PostgreSQL│      │ Key Vault  │     │   ACR    │         │
│    │+pgvector │      │ (Secrets)  │     │ (Images) │         │
│    │(Private) │      │            │     │          │         │
│    └──────────┘      └────────────┘     └──────────┘         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

Network Features:
• Azure CNI networking
• Network policies (pod-to-pod firewall)
• Network Security Groups
• Private DNS zones
• Service endpoints
```

## 🚀 Quick Start

### Prerequisites

```bash
# 1. Install tools
az --version          # Azure CLI
terraform --version   # Terraform >= 1.0
kubectl version       # kubectl
helm version          # Helm (optional)

# 2. Login to Azure
az login
az account set --subscription "<your-subscription-id>"
```

### Deploy Infrastructure

```bash
# 1. Navigate to AKS terraform directory
cd terraform/aks

# 2. Copy MVP configuration
cp terraform.tfvars.example terraform.tfvars

# Or use pre-configured environment
cp environments/mvp.tfvars terraform.tfvars

# 3. Edit configuration
nano terraform.tfvars

# Required: Set these values
# - db_password
# - voyage_api_key

# 4. Initialize Terraform
terraform init

# 5. Preview changes
terraform plan

# 6. Deploy (takes ~10-15 minutes)
terraform apply

# Save outputs
terraform output -json > outputs.json
```

### Build and Deploy Application

```bash
# 1. Get AKS credentials
az aks get-credentials \
  --resource-group $(terraform output -raw resource_group_name) \
  --name $(terraform output -raw aks_cluster_name)

# 2. Verify cluster access
kubectl get nodes
kubectl get namespaces

# 3. Build and push Docker image
ACR_NAME=$(terraform output -raw acr_name)
ACR_SERVER=$(terraform output -raw acr_login_server)

az acr login --name $ACR_NAME

cd ../..  # Back to project root
docker build -t ${ACR_SERVER}/qwelli:latest .
docker push ${ACR_SERVER}/qwelli:latest

# 4. Initialize database
DB_HOST=$(cd terraform/aks && terraform output -raw postgres_fqdn)
DB_PASS="<your-db-password>"

psql "postgresql://qwelliadmin:${DB_PASS}@${DB_HOST}:5432/qwelli?sslmode=require" \
  -f scripts/init-db.sql

# 5. Update Kubernetes manifests with your values
cd k8s

# Replace placeholders in files:
# - deployment.yaml: ${ACR_LOGIN_SERVER}, ${POSTGRES_FQDN}
# - secret.yaml: ${POD_IDENTITY_CLIENT_ID}, ${KEY_VAULT_NAME}, ${TENANT_ID}
# - serviceaccount.yaml: ${POD_IDENTITY_CLIENT_ID}

# Get values from terraform
cd ../terraform/aks
export ACR_LOGIN_SERVER=$(terraform output -raw acr_login_server)
export POSTGRES_FQDN=$(terraform output -raw postgres_fqdn)
export POD_IDENTITY_CLIENT_ID=$(terraform output -raw pod_identity_client_id)
export KEY_VAULT_NAME=$(terraform output -raw key_vault_name)
export TENANT_ID=$(terraform output -raw pod_identity_tenant_id)

# Update manifests (using sed or your editor)
cd ../../k8s
sed -i "s|\${ACR_LOGIN_SERVER}|${ACR_LOGIN_SERVER}|g" deployment.yaml
sed -i "s|\${POSTGRES_FQDN}|${POSTGRES_FQDN}|g" deployment.yaml configmap.yaml
sed -i "s|\${POD_IDENTITY_CLIENT_ID}|${POD_IDENTITY_CLIENT_ID}|g" secret.yaml serviceaccount.yaml
sed -i "s|\${KEY_VAULT_NAME}|${KEY_VAULT_NAME}|g" secret.yaml
sed -i "s|\${TENANT_ID}|${TENANT_ID}|g" secret.yaml

# 6. Deploy to Kubernetes
kubectl apply -f namespace.yaml
kubectl apply -f serviceaccount.yaml
kubectl apply -f configmap.yaml
kubectl apply -f secret.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f hpa.yaml
kubectl apply -f network-policy.yaml

# 7. Watch deployment
kubectl get pods -n qwelli -w

# 8. Get service URL
kubectl get service qwelli-service -n qwelli

# Access application
export SERVICE_IP=$(kubectl get service qwelli-service -n qwelli -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "Application URL: http://${SERVICE_IP}"
curl http://${SERVICE_IP}/health
```

## 📊 Cost Optimization Strategies

### MVP Phase (~$70-100/month)

```hcl
# terraform/aks/environments/mvp.tfvars
system_node_size      = "Standard_D2s_v3"  # $70/month
system_node_min_count = 1
user_node_size        = "Standard_D2s_v3"  # $70/month (on-demand)
user_node_min_count   = 1
user_node_max_count   = 3
db_sku_name           = "B_Standard_B1ms"  # $12/month
acr_sku               = "Basic"             # $5/month
```

**Cost Saving Tips:**
```bash
# Scale down when not in use
kubectl scale deployment qwelli --replicas=0 -n qwelli

# Scale back up
kubectl scale deployment qwelli --replicas=2 -n qwelli

# Or set minimum replicas to 0 in HPA (scale to zero when idle)
# Edit hpa.yaml: minReplicas: 0
```

### Growth Phase (~$300-500/month)

```hcl
user_node_size        = "Standard_D4s_v3"  # 4 vCPU, 16GB
user_node_min_count   = 2
user_node_max_count   = 8
db_sku_name           = "GP_Standard_D2s_v3"
```

### Production Phase (~$800-1,500/month)

```hcl
# Use production.tfvars
system_node_size      = "Standard_D4s_v3"
system_node_count     = 3  # HA
user_node_size        = "Standard_D8s_v3"
user_node_min_count   = 3
user_node_max_count   = 10
db_sku_name           = "GP_Standard_D4s_v3"
enable_private_cluster = true
```

**Additional Savings:**
- **Reserved Instances**: 30-40% discount (1-3 year commitment)
- **Spot Instances**: 60-90% discount (for non-critical pods)
- **Azure Hybrid Benefit**: Use existing Windows licenses

## 🔒 Security Features

### Network Security

1. **VNet Isolation**
   - All resources in private virtual network
   - No public IPs on pods
   - Private endpoints for database

2. **Network Policies**
   - Pod-to-pod firewall rules
   - Ingress/egress controls
   - Deny-all-by-default approach

3. **Azure CNI**
   - Each pod gets VNet IP
   - NSG rules apply to pods
   - Better integration with Azure services

### Identity & Access

1. **Workload Identity**
   - Pods authenticate to Key Vault without secrets
   - OIDC-based, no credentials in code
   - Automatic token rotation

2. **Azure AD RBAC**
   - Control who can access cluster
   - Role-based permissions
   - MFA support

3. **Key Vault Integration**
   - Secrets stored in Key Vault
   - Mounted as volumes in pods
   - Automatic secret rotation

### Pod Security

1. **Security Contexts**
   - Run as non-root user
   - Read-only root filesystem
   - Drop all capabilities

2. **Resource Limits**
   - CPU and memory limits
   - Prevent resource exhaustion
   - Quality of Service guarantees

## 📈 Monitoring & Observability

### Container Insights

```bash
# View pod metrics in Azure Portal
# Or query with Azure CLI
az monitor log-analytics query \
  --workspace $(terraform output -raw log_analytics_workspace_id) \
  --analytics-query "ContainerLog | where TimeGenerated > ago(1h) | limit 100"
```

### Application Logs

```bash
# View real-time logs
kubectl logs -f deployment/qwelli -n qwelli

# View logs from all pods
kubectl logs -f -l app=qwelli -n qwelli

# View logs from specific pod
kubectl logs <pod-name> -n qwelli
```

### Metrics

```bash
# Pod metrics
kubectl top pods -n qwelli

# Node metrics
kubectl top nodes

# HPA status
kubectl get hpa -n qwelli
```

### Health Checks

```bash
# Check pod health
kubectl get pods -n qwelli

# Describe pod for events
kubectl describe pod <pod-name> -n qwelli

# Test endpoints
kubectl exec -it <pod-name> -n qwelli -- curl localhost:8080/health
```

## 🔄 Scaling Operations

### Manual Scaling

```bash
# Scale deployment
kubectl scale deployment qwelli --replicas=5 -n qwelli

# Scale nodes
az aks scale \
  --resource-group <rg-name> \
  --name <aks-name> \
  --node-count 3 \
  --nodepool-name user
```

### Automatic Scaling

**Horizontal Pod Autoscaler (HPA)**
```yaml
# Already configured in k8s/hpa.yaml
# Scales pods based on CPU/memory
minReplicas: 2
maxReplicas: 10
targetCPUUtilization: 70%
```

**Cluster Autoscaler**
```bash
# Enabled by default in Terraform
# Automatically adds/removes nodes based on pod demands
```

## 🚨 Troubleshooting

### Pods Not Starting

```bash
# Check pod status
kubectl describe pod <pod-name> -n qwelli

# Common issues:
# - Image pull errors: Check ACR credentials
# - Key Vault access: Verify workload identity setup
# - Database connection: Check network policies and connection string
```

### Database Connection Issues

```bash
# Test from pod
kubectl exec -it <pod-name> -n qwelli -- sh
# Inside pod:
apt-get update && apt-get install -y postgresql-client
psql "host=$DB_HOST port=5432 dbname=qwelli user=qwelliadmin sslmode=require"
```

### Key Vault Access Issues

```bash
# Verify workload identity
kubectl describe serviceaccount qwelli-sa -n qwelli

# Check secret provider class
kubectl describe secretproviderclass qwelli-secrets -n qwelli

# View pod events
kubectl get events -n qwelli --field-selector involvedObject.name=<pod-name>
```

### Network Policy Issues

```bash
# Test connectivity
kubectl run test-pod --image=busybox -it --rm -n qwelli -- sh
# Inside pod:
wget -O- http://qwelli-internal:8080/health
```

## 🔄 Updates & Maintenance

### Update Application

```bash
# Build new version
docker build -t ${ACR_SERVER}/qwelli:v2 .
docker push ${ACR_SERVER}/qwelli:v2

# Update deployment
kubectl set image deployment/qwelli qwelli=${ACR_SERVER}/qwelli:v2 -n qwelli

# Watch rollout
kubectl rollout status deployment/qwelli -n qwelli

# Rollback if needed
kubectl rollout undo deployment/qwelli -n qwelli
```

### Update Infrastructure

```bash
cd terraform/aks

# Update terraform.tfvars with new values
nano terraform.tfvars

# Preview changes
terraform plan

# Apply updates
terraform apply
```

### Upgrade Kubernetes

```bash
# Check available versions
az aks get-upgrades --resource-group <rg-name> --name <aks-name>

# Upgrade (done via Terraform)
# Update kubernetes_version in terraform.tfvars
# Then: terraform apply
```

## 🗑️ Cleanup

### Delete Application

```bash
kubectl delete -f k8s/
```

### Delete Infrastructure

```bash
cd terraform/aks
terraform destroy
```

## 📚 Additional Resources

- [AKS Best Practices](https://docs.microsoft.com/azure/aks/best-practices)
- [Azure CNI Networking](https://docs.microsoft.com/azure/aks/configure-azure-cni)
- [Workload Identity](https://docs.microsoft.com/azure/aks/workload-identity-overview)
- [pgvector Documentation](https://github.com/pgvector/pgvector)

## 🆘 Support

For issues:
1. Check pod logs: `kubectl logs -f deployment/qwelli -n qwelli`
2. Check events: `kubectl get events -n qwelli`
3. Review [Troubleshooting](#-troubleshooting) section
4. Check Azure Portal for resource status

---

**Ready to deploy?** Start with `terraform/aks/environments/mvp.tfvars` for MVP, then scale up as needed!
