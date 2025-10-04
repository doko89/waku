# WAKU Deployment Guide

This guide covers various deployment options for WAKU WhatsApp API.

## Table of Contents

- [Docker Deployment](#docker-deployment)
- [Docker Compose Deployment](#docker-compose-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Production Considerations](#production-considerations)

## Docker Deployment

### Quick Start

```bash
# Pull the latest image
docker pull ghcr.io/YOUR_USERNAME/waku:latest

# Run the container
docker run -d \
  --name waku-api \
  -p 8080:8080 \
  -e API_TOKEN=your-super-secret-token-here \
  -e WEBHOOK_URL=https://your-webhook-endpoint.com/webhook \
  -e WEBHOOK_ENABLED=true \
  -v $(pwd)/sessions:/app/sessions \
  -v $(pwd)/temp:/app/temp \
  --restart unless-stopped \
  ghcr.io/YOUR_USERNAME/waku:latest
```

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `API_TOKEN` | Bearer token for API authentication | - | Yes |
| `PORT` | HTTP server port | `8080` | No |
| `SESSION_DIR` | Directory for session storage | `/app/sessions` | No |
| `TEMP_MEDIA_DIR` | Directory for temporary media files | `/app/temp` | No |
| `WEBHOOK_URL` | URL to send incoming messages | - | No |
| `WEBHOOK_ENABLED` | Enable webhook functionality | `false` | No |
| `WEBHOOK_RETRY` | Number of webhook retry attempts | `3` | No |
| `LOG_LEVEL` | Logging level (debug/info/warn/error) | `info` | No |

### Volume Mounts

**Important**: Always mount these directories to persist data:

- `/app/sessions` - WhatsApp session data (required for reconnection)
- `/app/temp` - Temporary media files (can be ephemeral)

## Docker Compose Deployment

### Basic Setup

1. **Create docker-compose.yml**:

```yaml
services:
  waku:
    image: ghcr.io/YOUR_USERNAME/waku:latest
    container_name: waku-api
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - API_TOKEN=${API_TOKEN}
      - WEBHOOK_URL=${WEBHOOK_URL}
      - WEBHOOK_ENABLED=true
      - LOG_LEVEL=info
    volumes:
      - ./sessions:/app/sessions
      - ./temp:/app/temp
    networks:
      - waku-network

networks:
  waku-network:
    driver: bridge
```

2. **Create .env file**:

```env
API_TOKEN=your-super-secret-token-here
WEBHOOK_URL=https://your-webhook-endpoint.com/webhook
```

3. **Start the service**:

```bash
docker compose up -d
```

### With Reverse Proxy (Nginx)

```yaml
services:
  waku:
    image: ghcr.io/YOUR_USERNAME/waku:latest
    container_name: waku-api
    restart: unless-stopped
    environment:
      - API_TOKEN=${API_TOKEN}
      - WEBHOOK_URL=${WEBHOOK_URL}
      - WEBHOOK_ENABLED=true
    volumes:
      - ./sessions:/app/sessions
      - ./temp:/app/temp
    networks:
      - waku-network

  nginx:
    image: nginx:alpine
    container_name: waku-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - waku
    networks:
      - waku-network

networks:
  waku-network:
    driver: bridge
```

**nginx.conf example**:

```nginx
events {
    worker_connections 1024;
}

http {
    upstream waku {
        server waku:8080;
    }

    server {
        listen 80;
        server_name your-domain.com;
        return 301 https://$server_name$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name your-domain.com;

        ssl_certificate /etc/nginx/ssl/cert.pem;
        ssl_certificate_key /etc/nginx/ssl/key.pem;

        location / {
            proxy_pass http://waku;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}
```

## Kubernetes Deployment

### Basic Deployment

1. **Create namespace**:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: waku
```

2. **Create secret**:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: waku-secrets
  namespace: waku
type: Opaque
stringData:
  api-token: "your-super-secret-token-here"
  webhook-url: "https://your-webhook-endpoint.com/webhook"
```

3. **Create PersistentVolumeClaim**:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: waku-sessions
  namespace: waku
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```

4. **Create Deployment**:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: waku
  namespace: waku
spec:
  replicas: 1  # Do not scale > 1 (session conflicts)
  selector:
    matchLabels:
      app: waku
  template:
    metadata:
      labels:
        app: waku
    spec:
      containers:
      - name: waku
        image: ghcr.io/YOUR_USERNAME/waku:latest
        ports:
        - containerPort: 8080
        env:
        - name: API_TOKEN
          valueFrom:
            secretKeyRef:
              name: waku-secrets
              key: api-token
        - name: WEBHOOK_URL
          valueFrom:
            secretKeyRef:
              name: waku-secrets
              key: webhook-url
        - name: WEBHOOK_ENABLED
          value: "true"
        - name: LOG_LEVEL
          value: "info"
        volumeMounts:
        - name: sessions
          mountPath: /app/sessions
        - name: temp
          mountPath: /app/temp
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /qr/healthcheck
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /qr/healthcheck
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: sessions
        persistentVolumeClaim:
          claimName: waku-sessions
      - name: temp
        emptyDir: {}
```

5. **Create Service**:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: waku
  namespace: waku
spec:
  selector:
    app: waku
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
```

6. **Create Ingress** (optional):

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: waku
  namespace: waku
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - waku.your-domain.com
    secretName: waku-tls
  rules:
  - host: waku.your-domain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: waku
            port:
              number: 8080
```

## Production Considerations

### Security

1. **Use Strong API Token**:
   ```bash
   # Generate a secure token
   openssl rand -base64 32
   ```

2. **Enable HTTPS**:
   - Use reverse proxy (Nginx, Traefik, Caddy)
   - Use Let's Encrypt for SSL certificates

3. **Firewall Rules**:
   - Only expose necessary ports
   - Whitelist trusted IPs if possible

4. **Regular Updates**:
   ```bash
   # Pull latest image
   docker pull ghcr.io/YOUR_USERNAME/waku:latest
   
   # Restart container
   docker compose down && docker compose up -d
   ```

### Monitoring

1. **Health Checks**:
   - Endpoint: `GET /qr/healthcheck`
   - Should return 404 (expected for non-existent session)

2. **Logs**:
   ```bash
   # Docker
   docker logs -f waku-api
   
   # Docker Compose
   docker compose logs -f
   
   # Kubernetes
   kubectl logs -f deployment/waku -n waku
   ```

3. **Metrics** (optional):
   - Integrate with Prometheus
   - Use Grafana for visualization

### Backup

**Important**: Always backup session data!

```bash
# Backup sessions directory
tar czf sessions-backup-$(date +%Y%m%d).tar.gz sessions/

# Restore sessions
tar xzf sessions-backup-20250104.tar.gz
```

### Scaling

**Warning**: Do NOT scale WAKU horizontally (replicas > 1)!

- WhatsApp sessions are device-specific
- Multiple instances will cause session conflicts
- Use vertical scaling (more CPU/RAM) instead

### Resource Requirements

**Minimum**:
- CPU: 250m (0.25 cores)
- Memory: 256Mi
- Storage: 1Gi (sessions)

**Recommended**:
- CPU: 500m (0.5 cores)
- Memory: 512Mi
- Storage: 10Gi (sessions)

**Per Device**:
- ~50-100MB RAM per active session
- ~10-50MB storage per session

### Troubleshooting

1. **Session not connecting**:
   - Check if QR code is being generated
   - Verify session files exist
   - Check logs for errors

2. **Webhook not working**:
   - Verify WEBHOOK_URL is accessible
   - Check webhook endpoint logs
   - Verify WEBHOOK_ENABLED=true

3. **Container keeps restarting**:
   - Check logs: `docker logs waku-api`
   - Verify environment variables
   - Check volume permissions

4. **Out of memory**:
   - Increase memory limits
   - Reduce number of active sessions
   - Check for memory leaks in logs

## Support

For issues and questions:
- GitHub Issues: https://github.com/YOUR_USERNAME/waku/issues
- Documentation: https://github.com/YOUR_USERNAME/waku/blob/main/README.md

