# SYSTEM CONTEXT: PROJECT "TALAS" 🍠

---
## [2026-02] Update: Reliability, Security, and Event-Driven Best Practices

- **RabbitMQ Reliability:**
    - user-service kini memiliki mekanisme auto-reconnect, channel recovery, dan event queueing pada utilitas RabbitMQ. Event tidak akan hilang meski RabbitMQ/channel sempat down; event akan di-queue dan dipublish ulang otomatis saat channel siap.

- **INTERNAL_SERVICE_SECRET Enforcement:**
    - Environment variable `INTERNAL_SERVICE_SECRET` sekarang wajib di-set di semua service (user, content, worker) dan sudah enforced di docker-compose. Semua internal API harus menggunakan header ini untuk autentikasi antar service.

- **Config Consistency:**
    - worker-service sekarang membaca `USER_SERVICE_URL` dan `CONTENT_SERVICE_URL` dari environment variable, memastikan integrasi internal API antar service lebih konsisten dan mudah dikonfigurasi.

- **Best Practice Event-Driven:**
    - Pada arsitektur event-driven, sangat penting untuk menerapkan event queueing dan retry. Implementasi pada user-service memastikan event tetap dikirim meski terjadi gangguan sementara pada broker/event bus.

---

## 1. Project Identity & Scope
* **Name:** Talas.
* **Concept:** "Threads x Dribbble". Platform showcase project untuk developer & desainer dengan fitur sosial microblogging.
* **Tagline:** "Where Makers Ship & Share".
* **Timeline:** 7 Days Hackathon (Focus on MVP & Core Features).
* **Repo Structure:** Monorepo.

## 2. Technical Stack & Infrastructure (Polyglot)
Sistem menggunakan arsitektur **Microservices** dengan **Strict Isolation**.

| Component | Technology | Port (Local) | Responsibility |
| :--- | :--- | :--- | :--- |
| **Frontend** | Next.js (TypeScript) | `3000` | Client UI & Upload Logic. |
| **Gateway** | Nginx | `80` | Reverse Proxy, SSL, **No-Cache Policy**. |
| **User Service** | Node.js (Express) + Prisma | `3001` | Auth, User Profile, Follows, Notif. |
| **Content Service** | Go (Fiber) + GORM | `8080` | Projects, Comments, Feed, Collaborations. |
| **Worker Service** | Go | - | Background Jobs (Consumer RabbitMQ). |
| **User DB** | PostgreSQL | `5432` | Data Users. |
| **Content DB** | PostgreSQL | `5433` | Data Projects & Interactions. |
| **Cache** | Redis | `6379` | Feed Timeline, **Trending Feed Cache**, Session. |
| **Search** | Elasticsearch | `9200` | Project Searching, Filtering, & **Trending Discovery**. |
| **Broker** | RabbitMQ | `5672` | Event Bus (**showcase.***, **comment.***, **collaborator.***). |
| **Storage** | Cloudinary | - | Media Storage (Direct Upload from Client). |

## 3. Critical Database Rules (The "Must-Follow")
1.  **UUID Strategy:**
    * SEMUA Primary Key (`id`) dan Foreign Key (`user_id`, `project_id`) **WAJIB UUID**.
    * DILARANG menggunakan Integer/Auto-increment.
2.  **Split Database (Isolation):**
    * **DB A (User):** Tables `users`, `user_credentials`, `refresh_tokens`, `follows`, `notifications`.
    * **DB B (Content):** Tables `showcases`, `showcase_media`, `categories`, `comments`, `showcase_likes`, `comment_likes`, `bookmarks`, `collaborators`.
3.  **The "Ghost FK" Rule:**
    * Di **Content DB**, kolom `user_id` disimpan sebagai `UUID` biasa + `INDEX`.
    * **TIDAK BOLEH** ada constraint `REFERENCES users(id)` di Content DB karena tabel `users` tidak ada di sana.
4.  **Collaborators Table:**
    * Wajib memiliki kolom: `status` (ENUM: 'pending', 'accepted', 'rejected') dan `expired_at` (Timestamp).
    * Logic: Row dibuat saat invite, user harus accept via UI. Jika melewati `expired_at`, invitation dianggap kadaluarsa.

## 4. API Routing & Terminology
**Global Prefix:** `/api`

### A. Auth Strategy (Cookie-Based)
* **Mechanism:** JWT disimpan dalam **HttpOnly Cookie**.
* **Login/Refresh Response:** Token **TIDAK** dikembalikan di JSON Body, melainkan via Header `Set-Cookie`.
* **Security:** Backend harus validasi `Cookie` header pada setiap request protected.

### B. Route Definitions (By Service)

#### 🔐 User Service (Express) -> Prefix `/api`
* `POST /auth/register`, `/auth/login`, `/auth/refresh`
* `POST /auth/logout` (Clear Cookie & Revoke Token)
* `GET /users/me`, `PATCH /users/me` (Private Profile)
* `GET /users/:username` (Public Profile)
* `POST /users/:id/follow` (Toggle Follow)
* `GET /notifications`, `GET /notifications/count`
* `PATCH /notifications/read` (**Batch Read: Array User ID**)

#### 🎨 Content Service (Go) -> Prefix `/api`
* **Showcases (Projects):**
    * `POST /showcases` (Create - Direct Publish).
    * `GET /showcases/user/:id` (User Profile Feed).
    * `GET /showcases/:slug` (Detail by **Slug OR UUID**).
    * `PATCH /showcases/:id` (Edit), `DELETE /showcases/:id`.
    * `POST /showcases/:id/like`, `/showcases/:id/save`.
* **Collaborations (Invites):**
    * `POST /showcases/:id/collaborators` (Invite User -> Status Pending).
    * `GET /collaborations/invitations` (List undangan masuk untuk user login).
    * `PATCH /collaborations/:id/response` (Action: Accept/Reject).
* **Comments (Flattened Nesting):**
    * `GET /showcases/:id/comments` (List).
    * `POST /showcases/:id/comments` (Create).
    * `DELETE /comments/:id` (Delete direct by ID).
* **Discovery:**
    * `GET /feed` (Home Timeline via Redis).
    * `GET /search` (Explore via Elastic - Keyword & Filters).
    * `GET /feeds/trending` (Discovery via Elastic - Weighted Score).

#### ⚙️ Internal API (Private - Docker Network Only)
* Prefix `/internal/*`
* **Security:** Wajib menyertakan Header `x-service-secret` yang nilainya sama dengan ENV variable `INTERNAL_SERVICE_SECRET`.
* `GET /internal/users/:id/followers` (Pagination supported - Fan-out Feed).
* `POST /internal/users/bulk` (Return Map O(1) - Enrich data author/collaborator).

## 5. Workflow & Logic Constraints

### A. Media Upload (Direct Upload Pattern)
1.  **Frontend** upload file fisik langsung ke **Cloudinary**.
2.  **Frontend** dapat URL (`https://res.cloudinary...`).
3.  **Frontend** kirim JSON ke Backend (`POST /showcases`).
4.  **Backend** HANYA menyimpan URL string.

### B. Collaborator Flow
1.  **Invite:** Owner project input username -> Backend resolve ID -> Insert tabel `collaborators` (Status: PENDING, Expired: +7 Hari).
2.  **Notify:** Worker kirim notifikasi ke user yang di-invite.
3.  **Action:** User buka menu "Invitations" -> Klik Accept/Reject -> Update Status di DB.
4.  **Display:** Nama collaborator hanya muncul di Project Detail jika Status == ACCEPTED.

### C. Event-Driven (RabbitMQ)
* **Exchange:** `talas.events` (Topic).
* **Routing Keys:**
    * `showcase.created` -> Worker index ke Elastic + Fan-out ke Redis followers.
    * `showcase.liked`, `showcase.unliked`, `showcase.viewed` -> Worker update counters di Elastic (Atomic).
    * `comment.created`, `comment.deleted` -> Worker update comment counters di Elastic (Atomic).
    * `collaborator.invited` -> Worker create Notification.
    * `collaborator.responded`, `collaborator.removed` -> Worker create Notification + Sync Elastic.

### D. Pagination Strategy (Cursor Based)
* **Mechanism:** Elasticsearch **Search After** (Stable Cursor).
* **Tie-breaker:** Semua sort wajib diakhiri dengan field `id: asc` untuk stabilitas cursor.
* **Request:** `?cursor=eyJ...&limit=10`.
* **Response:**
    ```json
    "meta": {
      "next_cursor": "...",
      "has_more": true,
      "limit": 10
    }
    ```
* **Offset/Page:** Diperbolehkan hanya untuk **Trending Feed** (Base64 Encoded Page Number).

## 6. Development Guidelines
* **JSON Case:** Gunakan `snake_case` untuk response (`user_id`, `expired_at`).
* **Validation:** Validasi input ketat di Backend sebelum masuk logic.
* **Documentation:** Update `API_CONTRACT.md` sebelum coding fitur baru.

---

## 7. Testing Strategy & Infrastructure
### A. Integration Testing (User Service)
* **Framework:** Jest + Supertest.
* **Database Strategy:** 
    * Menggunakan database terisolasi `talas_db_test` (Dockerized Postgres).
    * `tests/setup.js`: Otomatis sinkronisasi schema (`db push`) dan truncate table sebelum setiap tes.
* **Scope:** 
    * **Auth Module:** Register, Login, Logout, Refresh Token, Me.
    * **User Module:** Public Profile, Private Profile, Follow/Unfollow, Update Profile.
* **Execution:** `npm run test:integration` (Load `.env.test`).
* **Coverage:** 52 Test Cases (Positive, Negative, Edge Cases).

## 8. Collaborator Rules (Edge Cases)
* Collaborator bisa **leave sendiri** dari project.
* Owner **TIDAK BISA** kick/remove collaborator lain.
* Jika owner leave dan ada collaborator lain:
    * Ownership **berpindah** ke collaborator terbaru (yang paling akhir join).
* Jika owner adalah **satu-satunya pemilik** (Sole Owner):
    * Owner **TIDAK BISA leave**.
    * Diarahkan untuk **Archive** atau **Delete** showcase.

### B. Archive Feature (Soft Delete)
* Endpoint: `DELETE /showcases/:id`.
* Mekanisme: **Soft Delete** (`deleted_at` terisi). Data tidak hilang dari DB, tapi tidak muncul di list/feed publik.
* Asset Media (Cloudinary): **TIDAK** dihapus saat soft delete (untuk memungkinkan restore).
* Hanya **Owner** yang bisa melakukan aksi ini.

---

## 9. Like/Unlike Flow (Event-Driven)

### A. Frontend (Optimistic UI)
1. User klik tombol Like/Unlike.
2. UI **langsung berubah** (icon + counter) tanpa menunggu API.
3. Request `POST /api/showcases/:id/like` dikirim di background.
4. Jika API error → **Rollback** ke state sebelumnya + tampilkan Toast error.

### B. Backend API (Content Service)
1. Cek tabel `showcase_likes`:
    * Belum ada → `INSERT` (Like).
    * Sudah ada → `DELETE` (Unlike/Toggle).
2. Gunakan **Unique Constraint** `(user_id, showcase_id)` untuk mencegah double-insert.
3. Publish event ke RabbitMQ:
    * Exchange: `talas.events`
    * Routing Key: `project.liked` atau `project.unliked`
    * Payload: `{ project_id, user_id, author_id, timestamp }`
4. Response: `{ "is_liked": true/false }` (tanpa count, agar < 50ms).

### C. Worker (Async Side Effects)
1. **Update Counter:** `UPDATE showcases SET likes_count = likes_count +/- 1`.
2. **Update Redis Trending:** (Lihat Section 9).
3. **Sync Elasticsearch:** Update field `likes_count` di dokumen project.
4. **Send Notification:** Ke Author + semua Collaborators yang `ACCEPTED`.

---

### A. Implementation Flow (Elasticsearch + Redis)
1. **Repository Layer (ES)**: Menggunakan `function_score` dengan `script_score`.
   - **Formula**: `(Views * 1) + (Likes * 10) + (Comments * 30) + (Collaborators * 50)`.
   - **Filter**: HANYA mengambil showcase yang dibuat dalam **7 hari terakhir** (`now-7d/d`).
2. **Usecase Layer (Redis - Cache Aside)**:
   - Check Redis (`feeds:trending:{page}:{limit}`).
   - Jika Miss: Fetch dari ES -> Simpan ke Redis (TTL 10m) -> Return.
   - Jika Hit: Langsung return data cache.

**Benefit:** Algoritma trending yang dinamis namun tetap ringan berkat layer caching Redis.

---

## 11. Comment System Strategy (Recursive Tree + Tombstoning)

### Logic:
* **Storage:** Parent-Child relationship (`parent_id`).
* **Retrieval (Backend):** 
    * Mengambil semua komentar (termasuk Soft Deleted).
    * Membangun **Tree Structure** penuh secara rekursif.
    * **Tombstoning Logic (Pruning):**
        * Jika komentar `deleted` DAN punya anak (replies) yang aktif -> Konten diganti `"[Comment deleted]"` (Tombstone).
        * Jika komentar `deleted` DAN tidak punya anak aktif -> Dihapus dari response (Pruned).
* **Edit Tracking:**
    * Field `is_edited` (boolean) menandakan jika komentar (atau showcase) pernah diedit.
* **Frontend:** Menerima struktur tree JSON nested, merender sesuai kedalaman (indentasi).

---

## 12. Notification Batching (Smart Aggregation)

### Logic di Worker (saat menerima event Like):
1. Cek DB Notifikasi: Ada notifikasi tipe `LIKE` untuk `project_id` ini dengan status `UNREAD`?
2. **Jika ADA:**
    * Update teks: `"{User A}, {User B}, dan {N} lainnya menyukai showcase-mu"`.
    * Update `updated_at` agar naik ke paling atas list.
3. **Jika TIDAK ADA:**
    * Buat row notifikasi baru.

### Recipients:
* Notifikasi dikirim ke **Author (Owner)** DAN semua **Collaborators** dengan status `ACCEPTED`.
* Skip jika `liker_id == recipient_id` (tidak notif diri sendiri).