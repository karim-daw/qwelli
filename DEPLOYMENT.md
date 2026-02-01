# Qwelli Deployment Guide

Quick reference for deploying Qwelli in different environments.

## Quick Start (Local Development)

```bash
# 1. Copy environment template
cp .env.example .env

# 2. Edit .env and add your Voyage API key
nano .env

# 3. Start everything
docker-compose up

# Access at http://localhost:8080
```

## Production Deployment

### Prerequisites

1. **Managed PostgreSQL** with pgvector extension
2. **Server/VM** with Docker installed
3. **Voyage AI API key**

### Step 1: Set up PostgreSQL

#### Azure Database for PostgreSQL

```bash
# Create server
az postgres flexible-server create \
  --name qwelli-db \
  --resource-group mygroup \
  --location eastus \
  --admin-user qwelli \
  --admin-password <secure-password> \
  --sku-name Standard_B2s \
  --tier Burstable \
  --storage-size 32

# Enable pgvector
az postgres flexible-server parameter set \
  --resource-group mygroup \
  --server-name qwelli-db \
  --name azure.extensions \
  --value VECTOR

# Configure firewall
az postgres flexible-server firewall-rule create \
  --resource-group mygroup \
  --name qwelli-db \
  --rule-name allow-my-server \
  --start-ip-address <your-server-ip> \
  --end-ip-address <your-server-ip>

# Get connection string
az postgres flexible-server show-connection-string \
  --server-name qwelli-db \
  --admin-user qwelli \
  --admin-password <password> \
  --database-name qwelli
```

#### AWS RDS for PostgreSQL

```bash
# Create DB instance
aws rds create-db-instance \
  --db-instance-identifier qwelli-db \
  --db-instance-class db.t3.micro \
  --engine postgres \
  --engine-version 16.1 \
  --master-username qwelli \
  --master-user-password <secure-password> \
  --allocated-storage 20

# Create parameter group with pgvector
aws rds create-db-parameter-group \
  --db-parameter-group-name pgvector-params \
  --db-parameter-group-family postgres16 \
  --description "Parameter group for pgvector"

aws rds modify-db-parameter-group \
  --db-parameter-group-name pgvector-params \
  --parameters "ParameterName=shared_preload_libraries,ParameterValue=vector,ApplyMethod=pending-reboot"

# Apply parameter group
aws rds modify-db-instance \
  --db-instance-identifier qwelli-db \
  --db-parameter-group-name pgvector-params \
  --apply-immediately

# Reboot to apply
aws rds reboot-db-instance --db-instance-identifier qwelli-db
```

#### Other PostgreSQL Providers

- **DigitalOcean:** Managed Databases → PostgreSQL → Enable pgvector in extensions
- **Supabase:** pgvector included by default
- **Neon:** pgvector included by default
- **Self-hosted:** `CREATE EXTENSION vector;`

### Step 2: Deploy Application

```bash
# On your server
git clone <your-repo>
cd qwelli

# Create production .env
cat > .env << 'EOF'
DATABASE_URL=postgresql://qwelli:password@your-db-host:5432/qwelli?sslmode=require
VOYAGE_API_KEY=your_voyage_api_key
VOYAGE_MODEL=voyage-multimodal-3
PORT=8080
ENABLE_RERANKER=true
EOF

# Start with production config (no local postgres)
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d qwelli

# Check logs
docker-compose logs -f qwelli
```

### Step 3: Set Up Reverse Proxy (Optional but Recommended)

#### Using Caddy (Automatic HTTPS)

```bash
# Install Caddy
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install caddy

# Configure Caddy
sudo nano /etc/caddy/Caddyfile
```

Add:
```
qwelli.yourdomain.com {
    reverse_proxy localhost:8080
}
```

```bash
# Reload Caddy
sudo systemctl reload caddy
```

#### Using Nginx

```bash
# Install nginx
sudo apt install nginx

# Create config
sudo nano /etc/nginx/sites-available/qwelli
```

Add:
```nginx
server {
    listen 80;
    server_name qwelli.yourdomain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/qwelli /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

# Get SSL certificate
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d qwelli.yourdomain.com
```

## Updating Production

```bash
# Pull latest code
git pull

# Rebuild image
docker-compose build qwelli

# Restart with zero downtime
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d qwelli

# Check logs
docker-compose logs -f qwelli
```

## Monitoring

### Health Checks

```bash
# Liveness check
curl http://localhost:8080/health

# Should return: {"status": "ok"}
```

### View Logs

```bash
# Real-time logs
docker-compose logs -f qwelli

# Last 100 lines
docker-compose logs --tail 100 qwelli
```

### Resource Usage

```bash
# Container stats
docker stats qwelli-app

# PostgreSQL stats (if using docker-compose postgres)
docker stats qwelli-postgres
```

## Troubleshooting

### Container won't start

```bash
# Check logs
docker-compose logs qwelli

# Common issues:
# 1. Missing VOYAGE_API_KEY - add to .env
# 2. Can't connect to database - check DATABASE_URL
# 3. Database doesn't have pgvector - enable extension
```

### Database connection issues

```bash
# Test connection manually
psql "postgresql://user:pass@host:5432/qwelli?sslmode=require"

# Check if pgvector is installed
psql "postgresql://..." -c "SELECT * FROM pg_extension WHERE extname = 'vector';"

# If not, install it
psql "postgresql://..." -c "CREATE EXTENSION IF NOT EXISTS vector;"
```

### Can't access web UI

```bash
# Check if container is running
docker ps | grep qwelli

# Check if port is accessible
curl http://localhost:8080/health

# Check firewall rules
sudo ufw status
sudo ufw allow 8080/tcp  # If needed
```

## Backup and Recovery

### Backup Database

```bash
# Create backup
pg_dump "postgresql://user:pass@host:5432/qwelli?sslmode=require" > backup.sql

# Or for Azure
az postgres flexible-server backup create \
  --resource-group mygroup \
  --name qwelli-db \
  --backup-name "manual-$(date +%Y%m%d)"
```

### Restore Database

```bash
# Restore from backup
psql "postgresql://user:pass@host:5432/qwelli?sslmode=require" < backup.sql
```

## Cost Estimates

### Small Deployment (0-1k requests/day)
- **Database:** Standard_B1s / db.t3.micro (~$15-20/month)
- **Compute:** 1 CPU / 2GB VM (~$10-15/month)
- **Total:** ~$25-35/month

### Medium Deployment (1k-10k requests/day)
- **Database:** Standard_B2s / db.t3.small (~$30-40/month)
- **Compute:** 2 CPU / 4GB VM (~$20-30/month)
- **Total:** ~$50-70/month

### Large Deployment (10k+ requests/day)
- **Database:** Standard_D2s / db.t3.medium (~$100-150/month)
- **Compute:** 4 CPU / 8GB VM (~$40-60/month)
- **Load Balancer:** (~$20/month)
- **Total:** ~$160-230/month

## Security Checklist

- [ ] Use strong PostgreSQL password
- [ ] Enable SSL/TLS for database (sslmode=require)
- [ ] Don't commit .env file to git
- [ ] Use firewall to restrict database access
- [ ] Set up HTTPS with Caddy or Let's Encrypt
- [ ] Regularly update Docker images
- [ ] Enable automated backups
- [ ] Use secret management (Azure Key Vault, AWS Secrets Manager, etc.)
- [ ] Restrict API access with authentication (if needed)
- [ ] Monitor logs for suspicious activity

## Support

For issues or questions:
- Check the main [README.md](./README.md)
- Review [ARCHITECTURE.md](./ARCHITECTURE.md)
- Open an issue on GitHub
