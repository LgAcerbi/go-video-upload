# Production Deployment Plan — Oracle VPS

> **Target:** Deploy the entire video-processing platform on an Oracle Cloud VPS using
> Portainer with separate stacks, behind a reverse proxy with TLS on `lgacerbi.com`.
>
> **Public-facing (customer) endpoint:** `https://uploader.lgacerbi.com/api` — **only the upload service** is exposed as the public application.
>
> **Admin-only (your domain):** All other services (Portainer, MinIO, RabbitMQ, Adminer, Traefik) are reachable only via your bought domain `*.lgacerbi.com` and should be restricted (e.g. IP whitelist) so they are not open to the internet.

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Prerequisites](#2-prerequisites)
3. [Create the Oracle Cloud Instance (VPS)](#3-create-the-oracle-cloud-instance-vps)
4. [VPS Initial Setup](#4-vps-initial-setup)
5. [Install Docker & Docker Compose](#5-install-docker--docker-compose)
6. [Install & Configure Portainer](#6-install--configure-portainer)
7. [Create the Docker Network](#7-create-the-docker-network)
8. [Stack 1 — Infrastructure Services](#8-stack-1--infrastructure-services)
9. [Stack 2 — Application Services](#9-stack-2--application-services)
10. [Stack 3 — Reverse Proxy (Traefik)](#10-stack-3--reverse-proxy-traefik)
11. [DNS Configuration](#11-dns-configuration)
12. [Database Initialization](#12-database-initialization)
13. [Verify the Deployment](#13-verify-the-deployment)
14. [Security Hardening](#14-security-hardening)
15. [Maintenance & Operations](#15-maintenance--operations)
16. [Rollback Procedure](#16-rollback-procedure)

---

## 1. Architecture Overview

**Access model:**


| Visibility                   | Services                                                                   | Purpose                          |
| ---------------------------- | -------------------------------------------------------------------------- | -------------------------------- |
| **Public**                   | Upload API (`uploader.lgacerbi.com/api`) + S3 presigns (`s3.lgacerbi.com`) | Customer-facing application only |
| **Your domain only (admin)** | Portainer, MinIO console, RabbitMQ, Adminer, Traefik dashboard             | Management UIs — restrict by IP  |


```
Internet
  │
  │  HTTPS :443
  ▼
┌──────────────────────────────────────────────────────┐
│  Traefik (reverse proxy + auto-TLS via Let's Encrypt)│
│  ─ uploader.lgacerbi.com/api  → upload:8080   [PUBLIC]│
│  ─ s3.lgacerbi.com            → minio:9000    [PUBLIC - presign uploads only]
│  ─ portainer.lgacerbi.com     → portainer:9000       │  (admin, restrict by IP)
│  ─ minio.lgacerbi.com         → minio:9001 (console) │  (admin, restrict by IP)
│  ─ rabbitmq.lgacerbi.com      → rabbitmq:15672 (mgmt)│  (admin, restrict by IP)
│  ─ adminer.lgacerbi.com       → adminer:8080         │  (admin, restrict by IP)
│  ─ traefik.lgacerbi.com       → Traefik dashboard   │  (admin, restrict by IP)
└──────────────┬───────────────────────────────────────┘
               │  Docker network: vp-net
               │
  ┌────────────┼──────────────────────────────┐
  │  STACK: infra                             │
  │  ┌──────────┐ ┌──────────┐ ┌───────────┐ │
  │  │ PostgreSQL│ │ RabbitMQ │ │   MinIO   │ │
  │  │  :5432    │ │:5672/mgmt│ │:9000/9001 │ │
  │  └──────────┘ └──────────┘ └───────────┘ │
  │  ┌──────────┐ ┌──────────────────┐       │
  │  │ Adminer  │ │   minio-init     │       │
  │  │  :8080   │ │ (run-once)       │       │
  │  └──────────┘ └──────────────────┘       │
  └───────────────────────────────────────────┘
  ┌────────────────────────────────────────────┐
  │  STACK: app                                │
  │  ┌────────┐ ┌────────────┐ ┌────────────┐ ┌────────┐ ┌────────┐ │
  │  │ upload │ │orchestrator│ │  metadata  │ │segment │ │publish │ │
  │  │:8080   │ │ (worker)   │ │  (worker)  │ │(worker)│ │(worker)│ │
  │  │:9090   │ │            │ │            │ │        │ │        │ │
  │  └────────┘ └────────────┘ └────────────┘ └────────┘ └────────┘ │
  │  ┌────────────┐                                                    │
  │  │ transcode  │  (worker)                                          │
  │  └────────────┘                                                    │
  └────────────────────────────────────────────┘
```

### Service Inventory


| Service          | Type           | Ports (internal)         | Depends On                     |
| ---------------- | -------------- | ------------------------ | ------------------------------ |
| **PostgreSQL**   | Infrastructure | 5432                     | —                              |
| **RabbitMQ**     | Infrastructure | 5672, 15672              | —                              |
| **MinIO**        | Infrastructure | 9000 (API), 9001 (UI)    | —                              |
| **Adminer**      | Infrastructure | 8080                     | PostgreSQL                     |
| **minio-init**   | Infrastructure | —                        | MinIO                          |
| **upload**       | Application    | 8080 (HTTP), 9090 (gRPC) | PostgreSQL, RabbitMQ, MinIO    |
| **orchestrator** | Application    | — (worker only)          | RabbitMQ, upload (gRPC)        |
| **metadata**     | Application    | — (worker only)          | RabbitMQ, MinIO, upload (gRPC) |
| **transcode**    | Application    | — (worker only)          | RabbitMQ, MinIO, upload (gRPC) |
| **segment**      | Application    | — (worker only)          | RabbitMQ, MinIO, upload (gRPC) |
| **publish**      | Application    | — (worker only)          | RabbitMQ, MinIO, upload (gRPC) |


---

## 2. Prerequisites

Before starting, make sure you have:

- An Oracle Cloud VPS (ARM or AMD) with **at least 4 GB RAM / 2 OCPU** (the Always Free tier Ampere A1 with 4 OCPU / 24 GB RAM is ideal)
- SSH access to the VPS (key-based auth configured)
- The domain `lgacerbi.com` purchased and DNS management access
- Your project source code pushed to a Git remote (GitHub, etc.)
- Familiarity with basic Linux terminal operations

### Recommended VPS Specs


| Resource | Minimum                          | Recommended      |
| -------- | -------------------------------- | ---------------- |
| CPU      | 2 cores                          | 4 cores          |
| RAM      | 4 GB                             | 8 GB+            |
| Disk     | 50 GB                            | 100 GB+          |
| OS       | Ubuntu 22.04+ or Oracle Linux 8+ | Ubuntu 24.04 LTS |


---

## 3. Create the Oracle Cloud Instance (VPS)

This section walks you through creating a Compute Instance (VPS) in Oracle Cloud. In the Oracle Cloud Console, go to **Menu (☰) → Compute → Instances**, then click **Create instance**. The wizard is organized into four areas; follow each step below.

### 3.1 — Basic Information


| Field               | What to do                                                                                                                                                                                                                                                                                                     |
| ------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Name**            | Enter a name for the instance (e.g. `video-processing-vps` or `lgacerbi-app`). This is only a label.                                                                                                                                                                                                           |
| **Placement**       | Leave **Availability domain** as the default (or pick one). **Fault domain** can stay default.                                                                                                                                                                                                                 |
| **Image and shape** | Click **Edit** next to "Image and shape".                                                                                                                                                                                                                                                                      |
| **Image**           | Choose an OS image: **Ubuntu 22.04** or **Ubuntu 24.04** (recommended). Optionally **Oracle Linux 8** if you prefer.                                                                                                                                                                                           |
| **Shape**           | Click **Change shape**. For Always Free: select **Ampere** and choose a shape with **4 OCPU** and **24 GB memory** (e.g. VM.Standard.A1.Flex). For paid: **AMD** (e.g. VM.Standard.E4.Flex) with at least **2 OCPU** and **4 GB RAM**; 4 OCPU / 8 GB+ is better for this stack. Confirm with **Select shape**. |


After setting image and shape, continue. You can leave **Primary VNIC**, **Add SSH keys**, and **Boot volume** as default for now; the next steps cover them.

### 3.2 — Security (Add SSH keys)


| Field                             | What to do                                                                                                                                                                                                                       |
| --------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Add SSH keys**                  | You need an SSH key to connect to the instance. Choose one of the following.                                                                                                                                                     |
| **Generate a key pair for me**    | Oracle creates a key pair. **Download the private key** immediately — it will not be shown again. Save it (e.g. `~/.ssh/oracle-vps-key`) and set permissions: `chmod 600 ~/.ssh/oracle-vps-key`. Optionally save the public key. |
| **Upload public key file (.pub)** | Use this if you already have a key. Select your `.pub` file (e.g. from `~/.ssh/id_rsa.pub` or `id_ed25519.pub`).                                                                                                                 |
| **Paste public key**              | Paste the contents of your public key (one line, starts with `ssh-rsa` or `ssh-ed25519`).                                                                                                                                        |
| **No SSH keys**                   | Do **not** select this — you would not be able to log in to the instance.                                                                                                                                                        |


Without an SSH key you cannot log in; ensure you download or upload a key before clicking **Create**.

### 3.3 — Networking (Primary VNIC)

Configure the primary network interface (VNIC) and IP addressing.

**Primary VNIC**


| Field         | What to do                                                          |
| ------------- | ------------------------------------------------------------------- |
| **VNIC name** | Optional. Give a name (e.g. `lgacerbi-main-vnic`) or leave default. |


**Primary network — Virtual cloud network**


| Field                                     | What to do                                                                                |
| ----------------------------------------- | ----------------------------------------------------------------------------------------- |
| **Select existing virtual cloud network** | Use if you already have a VCN (e.g. from a previous instance).                            |
| **Create new virtual cloud network**      | Use for a new tenancy or first instance; OCI will create a VCN and default security list. |
| **Specify OCID**                          | Use only if you are provisioning via API/CLI and have the VCN OCID.                       |
| **Virtual cloud network compartment**     | When using existing VCN: choose the compartment (e.g. your root `lgacerbi (root)`).       |
| **Virtual cloud network**                 | When using existing: select the VCN (e.g. `vcn-20260307-1142`).                           |


**Subnet**


| Field                        | What to do                                                                                       |
| ---------------------------- | ------------------------------------------------------------------------------------------------ |
| **Select existing subnet**   | Use if your VCN already has a subnet.                                                            |
| **Create new public subnet** | Use for first-time setup so the instance can get a public IP and be reachable from the internet. |
| **Subnet compartment**       | When using existing: same compartment as VCN (e.g. `lgacerbi (root)`).                           |
| **Subnet**                   | When using existing: pick the subnet (e.g. `subnet-20260307-1141 (regional)`).                   |
| **Subnet IPv4 prefixes**     | Shown for the chosen subnet (e.g. `10.0.0.0/24`). No action needed.                              |


**Private IPv4 address assignment**


| Field                                         | What to do                                                                        |
| --------------------------------------------- | --------------------------------------------------------------------------------- |
| **Automatically assign private IPv4 address** | Recommended. Oracle assigns the next free address in the subnet.                  |
| **Manually assign private IPv4 address**      | Only if you need a fixed private IP; enter an unused address in the subnet range. |


**Public IPv4 address assignment**


| Field                                        | What to do                                                                                         |
| -------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| **Automatically assign public IPv4 address** | **Select this** so the instance is reachable from the internet (SSH, and later Traefik on 80/443). |
| *(Assign later)*                             | Avoid for this use case; you need a public IP from the start for SSH and web.                      |


**IPv6 address assignment**


| Field                                      | What to do                                                                                            |
| ------------------------------------------ | ----------------------------------------------------------------------------------------------------- |
| **Assign IPv6 address from subnet prefix** | Optional. Only works if the VCN and subnet have IPv6 enabled. You can skip for this deployment.       |
| *(Warning)*                                | If the UI says the VCN/subnet do not support IPv6, ignore IPv6 unless you enable it in the VCN first. |


**Advanced options (Networking)**


| Field                                              | What to do                                                                                                                                                                                    |
| -------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Use network security groups to control traffic** | Optional. You can skip and control access via the VCN Security List (see [4.3 — Configure the firewall](#43--configure-the-firewall-iptables--oracle-cloud)). Configure NSGs later if needed. |


**DNS record**


| Field                                  | What to do                                                                                                |
| -------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| **Assign a private DNS record**        | Optional; gives a DNS name inside the VCN (e.g. for private hostname resolution).                         |
| **Do not assign a private DNS record** | Fine to use if you only need the instance’s public IP and your own domain (e.g. `uploader.lgacerbi.com`). |


**Hostname / Fully qualified domain name**


| Field        | What to do                                                                                                                                                                                         |
| ------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Hostname** | Optional. If you assigned a private DNS record, the FQDN may look like `<hostname>.subnet03071143.vcn03071143.oraclevcn.com`. You can leave default or set a short name (e.g. `video-processing`). |


**Launch options**


| Option                                                              | When to use                                                                     |
| ------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| **Let Oracle Cloud Infrastructure choose the best networking type** | **Recommended.** OCI picks based on shape and image.                            |
| **Paravirtualized networking**                                      | General purpose (enterprise apps, microservices, small DBs).                    |
| **Hardware-assisted (SR-IOV) networking**                           | Low-latency (e.g. video streaming, real-time). Does not support live migration. |


Use **Let Oracle Cloud Infrastructure choose** unless you have a specific reason otherwise. Ensure the instance has **Automatically assign public IPv4 address** so you can SSH and serve HTTPS.

### 3.4 — Storage


| Field                        | What to do                                                                                                                                                              |
| ---------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Boot volume**              | Default is a single boot volume. Click **Show advanced options** only if you need encryption or a different performance tier.                                           |
| **Boot volume size**         | Set **Boot volume size (GB)** to at least **50 GB** (recommended **100 GB** for this platform so you have space for Docker images, volumes, and temporary video files). |
| **Block volumes (optional)** | Do not add extra block volumes unless you need a separate data disk. The boot volume is enough to start.                                                                |


Click **Create** at the bottom. The instance will be provisioned; wait until **State** is **Running**.

**After creation:**

1. Note the **Public IP address** shown on the instance details page — you will use it for SSH and for DNS in a later step.
2. **Linking your domain to this IP:** You do **not** link the host you bought (e.g. `lgacerbi.com` / `uploader.lgacerbi.com`) here. That is done later by creating DNS records that point your hostnames to this public IP. See [Section 11 — DNS Configuration](#11-dns-configuration) for the exact steps (A records, CNAME, etc.). You can do DNS after the instance is created and the stack is running.
3. (Optional) In **More actions**, you can **Create instance configuration** or **Create instance pool** for cloning; for a single VPS this is not required.
4. Proceed to [Section 4 — VPS Initial Setup](#4-vps-initial-setup) to connect via SSH and configure the server.

---

## 4. VPS Initial Setup

### 4.1 — Connect via SSH

```bash
ssh -i ~/.ssh/your-key ubuntu@<VPS_PUBLIC_IP>
```

### 4.2 — Update the system

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y curl git htop ufw
```

### 4.3 — Configure the firewall (iptables + Oracle Cloud)

Oracle Cloud has **two firewalls** — the OS-level firewall and the VCN Security List in the Oracle Cloud Console. You must open ports in **both**.

**OS-level (UFW):**

```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp    # HTTP (Traefik)
sudo ufw allow 443/tcp   # HTTPS (Traefik)
sudo ufw enable
```

> Do NOT expose database/broker ports (5432, 5672, 9000, etc.) to the internet.
> All inter-service communication stays on the internal Docker network.

**Oracle Cloud Console — VCN Ingress Rules:**

1. Go to **Networking → Virtual Cloud Networks → your VCN → Subnet → Security List**
2. Add **Ingress Rules** for:
  - Source CIDR: `0.0.0.0/0`, Protocol: TCP, Dest Port: **80**
  - Source CIDR: `0.0.0.0/0`, Protocol: TCP, Dest Port: **443**
3. SSH (port 22) should already be open from provisioning.

### 4.4 — Configure Oracle Linux iptables (if using Oracle Linux instead of Ubuntu)

Oracle Linux has `iptables` rules that block traffic even after Security List changes:

```bash
sudo iptables -I INPUT 6 -m state --state NEW -p tcp --dport 80 -j ACCEPT
sudo iptables -I INPUT 6 -m state --state NEW -p tcp --dport 443 -j ACCEPT
sudo netfilter-persistent save
```

---

## 5. Install Docker & Docker Compose

```bash
# Install Docker using the official convenience script
curl -fsSL https://get.docker.com | sudo sh

# Add your user to the docker group (avoids needing sudo)
sudo usermod -aG docker $USER

# Log out and back in, then verify
docker --version
docker compose version
```

> Docker Compose v2 is included as a plugin with modern Docker. If `docker compose`
> doesn't work, install the plugin: `sudo apt install docker-compose-plugin`.

---

## 6. Install & Configure Portainer

Portainer runs as a standalone container and manages your other stacks.

```bash
# Create a persistent volume for Portainer data
docker volume create portainer_data

# Run Portainer CE
docker run -d \
  --name portainer \
  --restart always \
  -p 9443:9443 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v portainer_data:/data \
  portainer/portainer-ce:lts
```

**Initial setup:**

1. Open Portainer in your browser:
   - **Direct:** `https://<VPS_PUBLIC_IP>:9443` (ensure port 9443 is allowed in the firewall).
   - **SSH tunnel (recommended if 9443 is closed):** Run `ssh -L 9443:localhost:9443 ubuntu@<VPS_IP>` in a terminal, then open **`https://localhost:9443`** (use **https**, not http — otherwise you'll get "Client sent an HTTP request to an HTTPS server"). Your browser may warn about the certificate; accept or proceed to continue.
2. Create the admin account (use a strong password)
3. Add your Docker environment when prompted:
   - **Environment type:** Choose **Socket**. (Portainer is on the same server as Docker and uses the mounted socket; ignore Edge Agent, Agent, API, and Standard for this single-host setup.)
   - **Endpoint:** It will show the local Docker socket. Name it e.g. **Local** or **primary**.
   - If asked **Docker Standalone vs Swarm:** choose **Standalone** (this plan uses Compose/stacks, not Swarm). If you later switch to Swarm, you can add or convert the environment.
   - Click **Connect**.

> Once Traefik is set up (Step 10), you'll access Portainer at `https://portainer.lgacerbi.com`
> and can close port 9443 from the public firewall.

### Deploying Stacks via Portainer

For each stack below, you have two options:

- **Option A — Git Repository:** In Portainer → Stacks → Add Stack → select "Repository", point to your Git repo, and specify the compose file path.
- **Option B — Web Editor:** Paste the compose file content directly into Portainer's stack editor.

**Recommended: Option A** — this way you can redeploy by clicking "Pull and Redeploy" whenever you push changes.

> **Which environment for stacks?** Use the **Local** (Standalone Docker) environment. Do not use a Swarm environment for the stacks in this plan — they are written for Docker Compose. If you have both "Local" and a Swarm endpoint (e.g. swarm-production-01), create all stacks on **Local**.

---

## 7. Create the Docker Network

All stacks need to share one external network so containers can communicate across stacks. You can create it from the Portainer UI or from the command line.

### From Portainer UI

1. In Portainer, select your **Local** environment (or the environment where the stacks will run).
2. Go to **Networks** in the left sidebar.
3. Click **Add network**.
4. Fill in:
   - **Name:** `vp-net`
   - **Driver:** `bridge` (default)
   - Leave **Create additional options** collapsed unless you need custom subnet/gateway.
5. Click **Create the network**.

The network is now available. Every stack that declares `networks: vp-net: { external: true }` will use it.

### From the command line (SSH)

```bash
docker network create vp-net
```

This network is referenced as `external: true` in every stack file.

---

## 8. Stack 1 — Infrastructure Services

> Portainer → Stacks → Add Stack → Name: `**infra**`

Create this file in your repo as `deploy/infra-stack.yml`, or paste it into Portainer's web editor.

```yaml
# deploy/infra-stack.yml
version: "3.9"

services:
  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-videopipe}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:?Set POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB:-videopipe}
    volumes:
      - postgres-data:/var/lib/postgresql/data
    networks:
      - vp-net
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-videopipe} -d ${POSTGRES_DB:-videopipe}"]
      interval: 5s
      timeout: 5s
      retries: 5
      start_period: 10s

  adminer:
    image: adminer:latest
    restart: unless-stopped
    environment:
      ADMINER_DEFAULT_SERVER: postgres
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - vp-net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.adminer.rule=Host(`adminer.lgacerbi.com`)"
      - "traefik.http.routers.adminer.entrypoints=websecure"
      - "traefik.http.routers.adminer.tls.certresolver=letsencrypt"
      - "traefik.http.services.adminer.loadbalancer.server.port=8080"

  rabbitmq:
    image: rabbitmq:3-management
    restart: unless-stopped
    environment:
      RABBITMQ_DEFAULT_USER: ${RABBITMQ_USER:-videopipe}
      RABBITMQ_DEFAULT_PASS: ${RABBITMQ_PASS:?Set RABBITMQ_PASS}
    volumes:
      - rabbitmq-data:/var/lib/rabbitmq
    networks:
      - vp-net
    healthcheck:
      test: ["CMD", "rabbitmq-diagnostics", "-q", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.rabbitmq.rule=Host(`rabbitmq.lgacerbi.com`)"
      - "traefik.http.routers.rabbitmq.entrypoints=websecure"
      - "traefik.http.routers.rabbitmq.tls.certresolver=letsencrypt"
      - "traefik.http.services.rabbitmq.loadbalancer.server.port=15672"

  minio:
    image: minio/minio:latest
    restart: unless-stopped
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER:?Set MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD:?Set MINIO_ROOT_PASSWORD}
      MINIO_API_CORS_ALLOW_ORIGIN: "https://uploader.lgacerbi.com"
    volumes:
      - minio-data:/data
    networks:
      - vp-net
    labels:
      # MinIO Console
      - "traefik.enable=true"
      - "traefik.http.routers.minio-console.rule=Host(`minio.lgacerbi.com`)"
      - "traefik.http.routers.minio-console.entrypoints=websecure"
      - "traefik.http.routers.minio-console.tls.certresolver=letsencrypt"
      - "traefik.http.routers.minio-console.service=minio-console"
      - "traefik.http.services.minio-console.loadbalancer.server.port=9001"
      # MinIO API (for presigned URL PUT from browser)
      - "traefik.http.routers.minio-api.rule=Host(`s3.lgacerbi.com`)"
      - "traefik.http.routers.minio-api.entrypoints=websecure"
      - "traefik.http.routers.minio-api.tls.certresolver=letsencrypt"
      - "traefik.http.routers.minio-api.service=minio-api"
      - "traefik.http.services.minio-api.loadbalancer.server.port=9000"

  minio-init:
    image: minio/mc:latest
    depends_on:
      - minio
    networks:
      - vp-net
    entrypoint: >
      /bin/sh -c "
      echo 'Waiting for MinIO...';
      sleep 10;
      mc alias set myminio http://minio:9000 $${MINIO_ROOT_USER} $${MINIO_ROOT_PASSWORD};
      n=0; max=60;
      until mc ready myminio 2>/dev/null; do
        n=$$(($$n + 1));
        if [ $$n -ge $$max ]; then echo 'MinIO not ready'; exit 1; fi;
        echo 'Waiting... ($$n/$$max)'; sleep 2;
      done;
      mc mb myminio/videos --ignore-existing;
      echo 'Bucket videos ready';
      exit 0;
      "
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}

volumes:
  postgres-data:
  rabbitmq-data:
  minio-data:

networks:
  vp-net:
    external: true
```

### Environment Variables for this Stack

In Portainer, when creating the stack, add these **environment variables** (or use a `.env` file):


| Variable              | Example Value              | Notes                          |
| --------------------- | -------------------------- | ------------------------------ |
| `POSTGRES_USER`       | `videopipe`                | Not `admin` in production      |
| `POSTGRES_PASSWORD`   | `<strong-random-password>` | **Generate a strong password** |
| `POSTGRES_DB`         | `videopipe`                |                                |
| `RABBITMQ_USER`       | `videopipe`                |                                |
| `RABBITMQ_PASS`       | `<strong-random-password>` |                                |
| `MINIO_ROOT_USER`     | `videopipe`                |                                |
| `MINIO_ROOT_PASSWORD` | `<strong-random-password>` | Min 8 chars                    |


> **Generate passwords** with: `openssl rand -base64 24`

### Reusing the same Postgres, RabbitMQ, InfluxDB, Grafana for more services later

You can deploy additional stacks (new apps) that all use the **same** Postgres, RabbitMQ, and optionally InfluxDB and Grafana.

**How it works**

- All stacks use the **same external network** (`vp-net`). Containers in any stack can reach infra by **service name**: `postgres`, `rabbitmq`, `minio`, and (if you add them) `influxdb`, `grafana`.
- New stack → same pattern: in the new stack’s compose file, set `networks: [vp-net]` and point your app’s config to those hostnames (e.g. `DATABASE_URL=postgresql://...@postgres:5432/...`, `RABBITMQ_HOST=rabbitmq`).

**Steps when adding a new app later**

1. Create a new stack in Portainer (e.g. `my-other-app`).
2. In its compose file:
   - Add `networks: [vp-net]` to every service that must talk to infra.
   - Add at the bottom: `networks: vp-net: { external: true }`.
3. Set env vars (or use the same secrets) so the new app uses `postgres`, `rabbitmq`, etc. as hosts. Use the same credentials as your existing stacks (or separate DB/broker users if you prefer).

**Adding InfluxDB and Grafana as shared services**

If you want one InfluxDB and one Grafana for all apps, add them to the **infra** stack (same `deploy/infra-stack.yml`) so they run on `vp-net` and are reachable as `influxdb` and `grafana`. Example (add to `services:` and a volume for InfluxDB):

```yaml
  influxdb:
    image: influxdb:2-alpine
    restart: unless-stopped
    volumes:
      - influxdb-data:/var/lib/influxdb2
    networks:
      - vp-net
    environment:
      DOCKER_INFLUXDB_INIT_MODE: setup
      DOCKER_INFLUXDB_INIT_USERNAME: admin
      DOCKER_INFLUXDB_INIT_PASSWORD: ${INFLUXDB_INIT_PASSWORD:?set}
      DOCKER_INFLUXDB_INIT_ORG: myorg
      DOCKER_INFLUXDB_INIT_BUCKET: default

  grafana:
    image: grafana/grafana:latest
    restart: unless-stopped
    volumes:
      - grafana-data:/var/lib/grafana
    networks:
      - vp-net
    environment:
      GF_SECURITY_ADMIN_PASSWORD: ${GRAFANA_ADMIN_PASSWORD:?set}
      GF_SERVER_ROOT_URL: https://grafana.lgacerbi.com
```

Add `influxdb-data:` and `grafana-data:` under `volumes:`. (Use `influxdb-data`, not `influx-data`, so the name matches the service volume reference.) New apps then use `http://influxdb:8086` and `http://grafana:3000` (and you can expose Grafana via Traefik if needed).

---

## 9. Stack 2 — Application Services

> Portainer → Stacks → Add Stack → Name: `**app`**

This stack builds your Go services from source. You need the code available on the VPS.

### 8.1 — Clone the Repository on the VPS

```bash
mkdir -p ~/projects
cd ~/projects
git clone https://github.com/LgAcerbi/go-video-upload.git
cd go-video-upload
```

### 8.2 — Create the Stack File

Create `deploy/app-stack.yml` in your repo:

```yaml
# deploy/app-stack.yml
version: "3.9"

services:
  upload:
    build:
      context: ..
      dockerfile: services/upload/Dockerfile
    restart: unless-stopped
    environment:
      PORT: "8080"
      GRPC_PORT: "9090"
      ENVIRONMENT: "production"
      OBJECT_STORAGE: "MINIO"
      S3_ENDPOINT: "http://minio:9000"
      S3_PRESIGN_ENDPOINT: "https://s3.lgacerbi.com"
      PLAYBACK_BASE_URL: "https://s3.lgacerbi.com"
      S3_REGION: "us-east-1"
      S3_BUCKET: "videos"
      AWS_ACCESS_KEY_ID: ${MINIO_ROOT_USER}
      AWS_SECRET_ACCESS_KEY: ${MINIO_ROOT_PASSWORD}
      RABBITMQ_HOST: "rabbitmq"
      RABBITMQ_PORT: "5672"
      RABBITMQ_USER: ${RABBITMQ_USER}
      RABBITMQ_PASS: ${RABBITMQ_PASS}
      DATABASE_URL: "postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}"
    networks:
      - vp-net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.upload.rule=Host(`uploader.lgacerbi.com`) && PathPrefix(`/api`)"
      - "traefik.http.routers.upload.entrypoints=websecure"
      - "traefik.http.routers.upload.tls.certresolver=letsencrypt"
      - "traefik.http.services.upload.loadbalancer.server.port=8080"
      - "traefik.http.middlewares.upload-strip.stripprefix.prefixes=/api"
      - "traefik.http.routers.upload.middlewares=upload-strip"

  orchestrator:
    build:
      context: ..
      dockerfile: services/orchestrator/Dockerfile
    restart: unless-stopped
    environment:
      UPLOAD_GRPC_TARGET: "upload:9090"
      RABBITMQ_HOST: "rabbitmq"
      RABBITMQ_PORT: "5672"
      RABBITMQ_USER: ${RABBITMQ_USER}
      RABBITMQ_PASS: ${RABBITMQ_PASS}
    networks:
      - vp-net

  metadata:
    build:
      context: ..
      dockerfile: services/metadata/Dockerfile
    restart: unless-stopped
    environment:
      UPLOAD_GRPC_TARGET: "upload:9090"
      RABBITMQ_HOST: "rabbitmq"
      RABBITMQ_PORT: "5672"
      RABBITMQ_USER: ${RABBITMQ_USER}
      RABBITMQ_PASS: ${RABBITMQ_PASS}
      S3_ENDPOINT: "http://minio:9000"
      S3_REGION: "us-east-1"
      S3_BUCKET: "videos"
      AWS_ACCESS_KEY_ID: ${MINIO_ROOT_USER}
      AWS_SECRET_ACCESS_KEY: ${MINIO_ROOT_PASSWORD}
    networks:
      - vp-net

  transcode:
    build:
      context: ..
      dockerfile: services/transcode/Dockerfile
    restart: unless-stopped
    environment:
      UPLOAD_GRPC_TARGET: "upload:9090"
      S3_ENDPOINT: "http://minio:9000"
      S3_REGION: "us-east-1"
      S3_BUCKET: "videos"
      AWS_ACCESS_KEY_ID: ${MINIO_ROOT_USER}
      AWS_SECRET_ACCESS_KEY: ${MINIO_ROOT_PASSWORD}
      RABBITMQ_HOST: "rabbitmq"
      RABBITMQ_PORT: "5672"
      RABBITMQ_USER: ${RABBITMQ_USER}
      RABBITMQ_PASS: ${RABBITMQ_PASS}
    networks:
      - vp-net

  segment:
    build:
      context: ..
      dockerfile: services/segment/Dockerfile
    restart: unless-stopped
    environment:
      UPLOAD_GRPC_TARGET: "upload:9090"
      S3_ENDPOINT: "http://minio:9000"
      S3_REGION: "us-east-1"
      S3_BUCKET: "videos"
      AWS_ACCESS_KEY_ID: ${MINIO_ROOT_USER}
      AWS_SECRET_ACCESS_KEY: ${MINIO_ROOT_PASSWORD}
      RABBITMQ_HOST: "rabbitmq"
      RABBITMQ_PORT: "5672"
      RABBITMQ_USER: ${RABBITMQ_USER}
      RABBITMQ_PASS: ${RABBITMQ_PASS}
    networks:
      - vp-net

  publish:
    build:
      context: ..
      dockerfile: services/publish/Dockerfile
    restart: unless-stopped
    environment:
      UPLOAD_GRPC_TARGET: "upload:9090"
      S3_ENDPOINT: "http://minio:9000"
      S3_REGION: "us-east-1"
      S3_BUCKET: "videos"
      AWS_ACCESS_KEY_ID: ${MINIO_ROOT_USER}
      AWS_SECRET_ACCESS_KEY: ${MINIO_ROOT_PASSWORD}
      RABBITMQ_HOST: "rabbitmq"
      RABBITMQ_PORT: "5672"
      RABBITMQ_USER: ${RABBITMQ_USER}
      RABBITMQ_PASS: ${RABBITMQ_PASS}
    networks:
      - vp-net

networks:
  vp-net:
    external: true
```

### 8.3 — Environment Variables for this Stack

These must match what you set in the `infra` stack:


| Variable              | Value                   |
| --------------------- | ----------------------- |
| `POSTGRES_USER`       | `videopipe`             |
| `POSTGRES_PASSWORD`   | *(same as infra stack)* |
| `POSTGRES_DB`         | `videopipe`             |
| `RABBITMQ_USER`       | `videopipe`             |
| `RABBITMQ_PASS`       | *(same as infra stack)* |
| `MINIO_ROOT_USER`     | `videopipe`             |
| `MINIO_ROOT_PASSWORD` | *(same as infra stack)* |


### 8.4 — Important: `S3_PRESIGN_ENDPOINT`

The upload service generates **presigned PUT URLs** that the client's browser uses directly.
In production, these URLs must point to a publicly-reachable MinIO endpoint.

That's why `S3_PRESIGN_ENDPOINT` is set to `https://s3.lgacerbi.com` — this is the
Traefik-routed, TLS-terminated domain that exposes MinIO's S3 API (port 9000).

The browser will `PUT` the video file directly to `https://s3.lgacerbi.com/videos/<id>/original`.

### 8.5 — Deploying with Portainer from Git

If using Portainer's Git-based deployment:

1. Stacks → Add Stack → **Repository**
2. Repository URL: `https://github.com/LgAcerbi/go-video-upload.git`
3. Compose path: `deploy/app-stack.yml`
4. Fill in environment variables
5. Deploy

For rebuilds after code changes: **Pull and Redeploy** button in Portainer.

---

## 10. Stack 3 — Reverse Proxy (Traefik)

> Portainer → Stacks → Add Stack → Name: `**proxy*`*

Traefik automatically discovers Docker containers via labels and provisions TLS certificates
from Let's Encrypt.

Create `deploy/proxy-stack.yml`:

```yaml
# deploy/proxy-stack.yml
version: "3.9"

services:
  traefik:
    image: traefik:v3.3
    restart: unless-stopped
    command:
      - "--api.dashboard=true"
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--providers.docker.network=vp-net"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      # Redirect all HTTP → HTTPS
      - "--entrypoints.web.http.redirections.entrypoint.to=websecure"
      - "--entrypoints.web.http.redirections.entrypoint.scheme=https"
      # Let's Encrypt
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge=true"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web"
      - "--certificatesresolvers.letsencrypt.acme.email=${ACME_EMAIL:?Set ACME_EMAIL}"
      - "--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - letsencrypt-data:/letsencrypt
    networks:
      - vp-net
    labels:
      # Traefik Dashboard
      - "traefik.enable=true"
      - "traefik.http.routers.dashboard.rule=Host(`traefik.lgacerbi.com`)"
      - "traefik.http.routers.dashboard.entrypoints=websecure"
      - "traefik.http.routers.dashboard.tls.certresolver=letsencrypt"
      - "traefik.http.routers.dashboard.service=api@internal"
      # Basic auth for dashboard (generate with: htpasswd -nB admin)
      - "traefik.http.routers.dashboard.middlewares=dashboard-auth"
      - "traefik.http.middlewares.dashboard-auth.basicauth.users=${TRAEFIK_DASHBOARD_AUTH}"

  # Expose Portainer through Traefik (optional — requires Portainer on vp-net)
  # If Portainer was started standalone, connect it to vp-net first:
  #   docker network connect vp-net portainer

volumes:
  letsencrypt-data:

networks:
  vp-net:
    external: true
```

### Environment Variables


| Variable                 | Value                            | Notes                              |
| ------------------------ | -------------------------------- | ---------------------------------- |
| `ACME_EMAIL`             | `your-email@example.com`         | Let's Encrypt registration email   |
| `TRAEFIK_DASHBOARD_AUTH` | `admin:$2y$05$...` (bcrypt hash) | Generate with `htpasswd -nB admin` |


### Connect Portainer to `vp-net`

So Traefik can route to Portainer:

```bash
docker network connect vp-net portainer
```

Then add these labels to the running Portainer container, or create a small override. The
simplest approach is to add Portainer to the proxy stack:

Add to `proxy-stack.yml` under `services`:

```yaml
  portainer:
    image: portainer/portainer-ce:lts
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - portainer_data:/data
    networks:
      - vp-net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.portainer.rule=Host(`portainer.lgacerbi.com`)"
      - "traefik.http.routers.portainer.entrypoints=websecure"
      - "traefik.http.routers.portainer.tls.certresolver=letsencrypt"
      - "traefik.http.services.portainer.loadbalancer.server.port=9000"
```

And add `portainer_data:` to the `volumes:` section.

> If you do this, remove the standalone Portainer container created in Step 5:
> `docker stop portainer && docker rm portainer`

---

## 11. DNS Configuration

All subdomains use your bought domain `lgacerbi.com` and point to the same VPS. **Only the upload API and S3 presign endpoint are intended for public access;** the rest are admin UIs and should be protected with IP whitelisting in Traefik (see §13.2).

In your DNS provider (wherever you registered `lgacerbi.com`), create these **A records**,
all pointing to your VPS public IP:


| Type | Name        | Value             | TTL |
| ---- | ----------- | ----------------- | --- |
| A    | `@`         | `<VPS_PUBLIC_IP>` | 300 |
| A    | `uploader`  | `<VPS_PUBLIC_IP>` | 300 |
| A    | `s3`        | `<VPS_PUBLIC_IP>` | 300 |
| A    | `minio`     | `<VPS_PUBLIC_IP>` | 300 |
| A    | `rabbitmq`  | `<VPS_PUBLIC_IP>` | 300 |
| A    | `portainer` | `<VPS_PUBLIC_IP>` | 300 |
| A    | `traefik`   | `<VPS_PUBLIC_IP>` | 300 |
| A    | `adminer`   | `<VPS_PUBLIC_IP>` | 300 |


> **Alternative:** Use a single wildcard record `*.lgacerbi.com → <VPS_PUBLIC_IP>` to
> cover all subdomains at once. Wildcard certificates require DNS challenge instead
> of HTTP challenge in Traefik — the HTTP challenge approach above is simpler.

### Verify DNS propagation

```bash
dig uploader.lgacerbi.com +short
# Should return your VPS IP
```

---

## 12. Database Initialization

After the `infra` stack is up and PostgreSQL is healthy, run the schema migration.

### Option A — From the VPS

```bash
# Copy the SQL file to the VPS (if not already there via git clone)
scp scripts/create_tables.sql ubuntu@<VPS_IP>:~/

# Run it
docker exec -i $(docker ps -qf "name=infra-postgres") \
  psql -U videopipe -d videopipe < ~/create_tables.sql
```

### Option B — Via Adminer

1. Open `https://adminer.lgacerbi.com`
2. Login: Server=`postgres`, User=`videopipe`, Pass=`<your-password>`, DB=`videopipe`
3. Go to **SQL Command** and paste the contents of `scripts/create_tables.sql`
4. Execute

### Verify Tables

```bash
docker exec -it $(docker ps -qf "name=infra-postgres") \
  psql -U videopipe -d videopipe -c "\dt"
```

Expected output:

```
          List of relations
 Schema |      Name       | Type  |  Owner
--------+-----------------+-------+-----------
 public | uploads         | table | videopipe
 public | upload_steps    | table | videopipe
 public | video_renditions| table | videopipe
 public | videos          | table | videopipe
```

---

## 13. Verify the Deployment

### 12.1 — Check all containers are running

```bash
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
```

All services should show `Up` with healthy status where applicable.

### 12.2 — Test the upload API endpoint

```bash
# Health check (should get a response, even if 404 — means the service is reachable)
curl -s -o /dev/null -w "%{http_code}" https://uploader.lgacerbi.com/api/videos/upload/presign

# Test presign endpoint
curl -X POST https://uploader.lgacerbi.com/api/videos/upload/presign \
  -H "Content-Type: application/json" \
  -d '{"user_id":"00000000-0000-0000-0000-000000000001","title":"test"}'
```

You should get a JSON response with `upload_url` and `video_id`.

### 12.3 — Test infrastructure UIs (admin-only, from your IP)

From an IP that you've whitelisted, verify:

- **Portainer:** `https://portainer.lgacerbi.com`
- **RabbitMQ Management:** `https://rabbitmq.lgacerbi.com`
- **MinIO Console:** `https://minio.lgacerbi.com`
- **Adminer:** `https://adminer.lgacerbi.com`
- **Traefik Dashboard:** `https://traefik.lgacerbi.com`

These should be unreachable (e.g. 403) from other IPs if IP whitelist is applied.

### 12.4 — Test TLS

```bash
curl -vI https://uploader.lgacerbi.com 2>&1 | grep -E "subject:|issuer:|expire"
```

Should show a Let's Encrypt certificate.

---

## 14. Security Hardening

### 13.1 — Change all default credentials

This plan already uses environment variables instead of hardcoded `admin/admin`. Never
reuse the development credentials from `build/docker-compose.yml`.

### 13.2 — Restrict management UIs (admin-only, your domain)

**Only the upload service is public.** All other services are reachable via your domain but must be restricted so they are not open to the internet. Use IP whitelisting in Traefik for every admin UI (Portainer, MinIO console, RabbitMQ, Adminer, Traefik dashboard):

```yaml
# Add to the labels of adminer, rabbitmq, minio-console, portainer, traefik dashboard.
# Use your home/office IP or VPN IP so only you can access them.
- "traefik.http.middlewares.ip-whitelist.ipwhitelist.sourcerange=<YOUR_HOME_IP>/32"
- "traefik.http.routers.adminer.middlewares=ip-whitelist"
```

Apply the same `ip-whitelist` middleware to: `adminer`, `rabbitmq`, `minio-console`, `portainer`, `dashboard` (Traefik). Do **not** add this middleware to the `upload` or `minio-api` (s3) routers — those remain public for your application and presigned uploads.

### 13.3 — Disable Swagger in production

The upload service currently mounts `/docs/`* for Swagger. In production, either:

- Set `ENVIRONMENT=production` (already done — but the docs route is still registered)
- Add a build tag or env check to skip `r.Get("/docs/*", ...)` in production

### 13.4 — MinIO CORS in production

`MINIO_API_CORS_ALLOW_ORIGIN` is set to `https://uploader.lgacerbi.com` instead of `*`.
If you later have a frontend on a different domain, add it to this list.

### 13.5 — Enable Docker log rotation

Add to `/etc/docker/daemon.json`:

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
```

Then restart Docker: `sudo systemctl restart docker`

### 13.6 — Automatic security updates

```bash
sudo apt install unattended-upgrades
sudo dpkg-reconfigure -plow unattended-upgrades
```

---

## 15. Maintenance & Operations

### 14.1 — Updating Application Services

After pushing code changes to your Git remote:

**Via Portainer:**

1. Go to Stacks → `app`
2. Click **Pull and Redeploy** (if using Git-based stack)
3. Or manually: Editor → Update Stack → tick "Re-pull and redeploy"

**Via CLI on VPS:**

```bash
cd ~/projects/go-video-upload
git pull origin main
docker compose -f deploy/app-stack.yml build --no-cache
docker compose -f deploy/app-stack.yml up -d
```

### 14.2 — Viewing Logs

```bash
# All app service logs
docker compose -f deploy/app-stack.yml logs -f

# Single service
docker logs -f $(docker ps -qf "name=app-upload")
```

Or use Portainer → Containers → select container → Logs.

### 14.3 — Backups

**PostgreSQL:**

```bash
# Dump
docker exec $(docker ps -qf "name=infra-postgres") \
  pg_dump -U videopipe videopipe > backup_$(date +%Y%m%d).sql

# Restore
docker exec -i $(docker ps -qf "name=infra-postgres") \
  psql -U videopipe -d videopipe < backup_20260315.sql
```

**MinIO (video files):**

```bash
# Use mc CLI to sync to a backup location
docker run --rm --network vp-net minio/mc \
  alias set prod http://minio:9000 videopipe <password> && \
  mc mirror prod/videos /backup/videos
```

### 14.4 — Monitoring (Optional but Recommended)

Add a lightweight monitoring stack later:

- **cAdvisor** for container metrics
- **Prometheus** + **Grafana** for dashboards
- **Loki** for centralized log aggregation

---

## 16. Rollback Procedure

If a deployment goes wrong:

### Quick rollback (app stack only)

```bash
cd ~/projects/go-video-upload

# Check the previous working commit
git log --oneline -5

# Reset to last known good commit
git checkout <commit-hash>

# Rebuild and redeploy
docker compose -f deploy/app-stack.yml build
docker compose -f deploy/app-stack.yml up -d
```

### Nuclear option (full reset of app stack)

```bash
docker compose -f deploy/app-stack.yml down
docker compose -f deploy/app-stack.yml up -d --build
```

> **Never** `down -v` the infra stack unless you have backups — this deletes database
> and object storage volumes.

---

## Deployment Checklist

Use this as a final walkthrough before going live:

- VPS provisioned and SSH access working
- Firewall open for ports 80 and 443 only (both OS-level and Oracle Security List)
- Docker and Docker Compose installed
- `docker network create vp-net` executed
- DNS A records created for all subdomains → VPS IP
- DNS propagation verified (`dig uploader.lgacerbi.com`)
- `proxy` stack deployed — Traefik running, HTTP→HTTPS redirect working
- `infra` stack deployed — Postgres, RabbitMQ, MinIO all healthy
- Database schema applied (`create_tables.sql`)
- `app` stack deployed — upload, orchestrator, metadata running
- `https://uploader.lgacerbi.com/api/videos/upload/presign` returns valid JSON
- Presigned URL domain (`s3.lgacerbi.com`) resolves and TLS works
- **Only upload + s3 are public;** all management UIs (Portainer, MinIO, RabbitMQ, Adminer, Traefik) restricted by IP whitelist on your domain
- All management UIs accessible (from your IP) and protected
- Strong passwords set for all services (no `admin/admin`)
- Docker log rotation configured
- Backup strategy in place for PostgreSQL and MinIO

---

## Subdomain Summary


| Subdomain                   | Routes To                  | Visibility          | Purpose                         |
| --------------------------- | -------------------------- | ------------------- | ------------------------------- |
| `uploader.lgacerbi.com/api` | upload service :8080       | **Public**          | Customer-facing Upload REST API |
| `s3.lgacerbi.com`           | MinIO S3 API :9000         | **Public**          | Presigned URL file uploads      |
| `portainer.lgacerbi.com`    | Portainer :9000            | Your domain (admin) | Container management            |
| `minio.lgacerbi.com`        | MinIO Console :9001        | Your domain (admin) | Object storage admin            |
| `rabbitmq.lgacerbi.com`     | RabbitMQ Management :15672 | Your domain (admin) | Message broker admin            |
| `adminer.lgacerbi.com`      | Adminer :8080              | Your domain (admin) | Database admin                  |
| `traefik.lgacerbi.com`      | Traefik Dashboard          | Your domain (admin) | Proxy admin                     |


