# Talas 🍠

**Where Makers Ship & Share.**
Platform showcase project yang menggabungkan konsep *microblogging* (Threads) dengan *visual gallery* (Dribbble).

## 🏗 Tech Stack

Sistem dibangun menggunakan arsitektur **Microservices (Polyglot)**:

* **Frontend:** Next.js (TypeScript)
* **API Gateway:** Nginx (Port `80`)
* **User Service:** Express.js + PostgreSQL (Port `3001` - Secured)
* **Content Service:** Go + PostgreSQL (Port `8080` - Secured)
* **Worker Service:** Go (Background Jobs)
* **Infra:** Docker Compose, Redis, RabbitMQ, Elasticsearch

## 🚀 Getting Started

### 1. Prerequisite
* Docker & Docker Compose
* Node.js v20+ (untuk development local)
* Go v1.21+ (untuk development local)

### 2. Run via Docker Compose (Recommended)
Jalankan seluruh stack (Database, Backend, Gateway) sekaligus:

```bash
docker-compose up -d --build
```

Akses API via Gateway: `http://localhost:80/api/...`

### 3. Run Manually (Local Dev)
Jika ingin menjalankan service secara individual (contoh: User Service):

```bash
# Masuk ke direktori service
cd user-service

# Install dependencies
npm install

# Setup Environment (.env)
cp .env.example .env

# Jalankan Database (jika belum ada)
# Pastikan PostgreSQL & RabbitMQ aktif

# Jalankan Migrasi & Seeder
npx prisma migrate dev
npx prisma db seed

# Jalankan App
npm run dev
```

## 🔐 Default Credentials (Seeder)

Setelah menjalankan `npx prisma db seed`, gunakan akun ini untuk login/testing:

| Role | Email | Password |
| :--- | :--- | :--- |
| **Admin/User** | `useradmin@example.com` | `password` |

> Default user lain (user0 - user19) memiliki password default: `password123`

## 📂 Project Structure

```text
talas-app/
├── /frontend          # Next.js Client
├── /user-service      # Auth & User Mgmt (Express)
├── /content-service   # Project, Feed, & Search (Golang)
├── /nginx             # API Gateway Config
├── /docs              # Dokumentasi Teknis (API Contract, Context)
└── docker-compose.yml # Orchestration
```