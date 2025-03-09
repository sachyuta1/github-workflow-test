# 🚀 Deployment Guide 🌟

## 🔍 Prerequisites Checklist
Before diving into the deployment, ensure you have the following essentials ready:

✅ **Kubernetes Cluster** – A running Kubernetes (K8s) cluster. 🏷️

✅ **Persistent Volume Storage** – A storage solution like Rook, Longhorn, or Ceph must be deployed & running. 📂

🔎 **Verify:**
```sh
kubectl cluster-info
kubectl get pv
kubectl get namespace
```

---

## 🏷️ Essential Services Setup
Make sure the following services are up and running:

| Service  | Description | Namespace |
|----------|-------------|------------|
| 🔒 **Cert Manager** | Automated SSL/TLS certificate management | cert-manager |
| 🐄 **MinIO** | Object storage for handling file uploads | 1view |
| 💤 **NATS** | Message broker for event-driven communication | 1view |
| ⚡ **Redis** | Caching and session management | 1view |
| 🐅 **PostgreSQL** | Database for storing application data | 1view |

---

## 🚀 Step-by-Step Deployment Guide
We will deploy each component using Helm charts.

### 📌 Step 1: Deploy Cert Manager 🐄
```sh
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm install cert-manager jetstack/cert-manager -n cert-manager --create-namespace \
  --set installCRDs=true
```
🔎 **Verify:** `kubectl get pods -n cert-manager | grep cert-manager`

---

### 📌 Step 2: Create Environment Variables Script 🎯
```sh
#!/bin/bash

# Set MinIO credentials
export MINIO_ACCESS_KEY="admin"
export MINIO_SECRET_KEY="admin123"

# Set NATS credentials
export NATS_USERNAME="my-nats"
export NATS_PASSWORD="password123"

# Set Redis password
export REDIS_PASSWORD="my-redis123"

# Set PostgreSQL credentials
export DATABASE_USER="my-postgres"
export DATABASE_PASSWORD="mysecretpassword"
export DATABASE_NAME="postgres"
```
🔎 **Verify:** Check if the variables are set correctly: 'echo $env'
 
---

### 📌 Step 3: Deploy MinIO 🐄
```sh
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
helm install my-minio bitnami/minio -n 1view
  --set MINIO_ACCESS_KEY="$MINIO_ACCESS_KEY"   
  --set MINIO_SECRET_KEY="$MINIO_SECRET_KEY" 
```
🔎 **Verify:** `kubectl get pods -n 1view | grep minio`

---

### 📌 Step 4: Deploy NATS 💤
```sh
helm repo add nats https://nats-io.github.io/k8s/helm/charts/
helm repo update
helm install nats nats/nats -n 1view
  --set NATS_USERNAME="$NATS_USERNAME"
  --set NATS_PASSWORD="$NATS_PASSWORD"
```
🔎 **Verify:** `kubectl get pods -n 1view | grep nats`

---

### 📌 Step 5: Deploy Redis ⚡
```sh
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
helm install redis bitnami/redis -n 1view
  --set REDIS_PASSWORD="$REDIS_PASSWORD"
```
🔎 **Verify:** `kubectl get pods -n 1view | grep redis`

---

### 📌 Step 6: Deploy PostgreSQL 🐅
```sh
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
helm install my-postgresql bitnami/postgresql -n 1view
  --set DATABASE_USER="$DATABASE_USER"
  --set DATABASE_PASSWORD="$DATABASE_PASSWORD"
  --set DATABASE_NAME="$DATABASE_NAME"
```
🔎 **Verify:** `kubectl get pods -n 1view | grep postgres`

---

### 📌 Step 7: Deploy the Application 🚀
```sh
helm install test ./project-management-api/   --namespace 1view  
  --set DATABASE_USER="$DATABASE_USER"   
  --set NATS_USERNAME="$NATS_USERNAME"
  --set DATABASE_NAME="$DATABASE_NAME"   
  --set DATABASE_PASSWORD="$DATABASE_PASSWORD"   
  --set REDIS_PASSWORD="$REDIS_PASSWORD"   
  --set NATS_PASSWORD="$NATS_PASSWORD"   
  --set MINIO_ACCESS_KEY="$MINIO_ACCESS_KEY"   
  --set MINIO_SECRET_KEY="$MINIO_SECRET_KEY"
```
🔎 **Verify:** `kubectl get pods -n 1view | grep project-management-api`

---

## 🔍 Post Deployment Verification 🕵️
After deploying, confirm that everything is running smoothly:
```sh
kubectl get pods -n 1view
kubectl get svc -n 1view
```
📝 **Check Logs:**
```sh
kubectl logs -f <pod-name> -n 1view
```

---

## 🟢 Cleanup & Uninstallation 🧹
To remove all deployed services, run:
```sh
helm uninstall project-management-api -n 1view
helm uninstall postgres -n 1view
helm uninstall redis -n 1view
helm uninstall nats -n 1view
helm uninstall minio -n 1view
```
🔎 **Verify Cleanup:**
```sh
kubectl get pods -n 1view
kubectl get svc -n 1view
```

---

🎉 **Deployment Complete!** Enjoy your fully operational system! 🚀

