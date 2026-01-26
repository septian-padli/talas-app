# Talas 🍠

**Where Makers Ship & Share.**
Platform showcase project yang menggabungkan konsep *microblogging* (Threads) dengan *visual gallery* (Dribbble).

## 🏗 Tech Stack

Sistem dibangun menggunakan arsitektur **Microservices (Polyglot)**:

* **Frontend:** Next.js (TypeScript)
* **API Gateway:** Nginx
* **User Service:** Express.js (Node.js) + PostgreSQL
* **Content Service:** Go (Fiber) + PostgreSQL
* **Worker Service:** Go (Background Jobs)
* **Infra:** Docker Compose, Redis, RabbitMQ, Elasticsearch

## 📂 Project Structure

```text
talas-app/
├── /frontend          # Next.js Client
├── /user-service      # Auth & User Mgmt (Express)
├── /content-service   # Project, Feed, & Search (Golang)
├── /nginx             # API Gateway Config
├── /docs              # Dokumentasi Teknis
└── docker-compose.yml # Orchestration