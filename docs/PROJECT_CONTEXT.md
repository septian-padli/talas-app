# SYSTEM CONTEXT: PROJECT "TALAS" 🍠

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
| **Cache** | Redis | `6379` | Feed Timeline (ZSET), Session, Counters. |
| **Search** | Elasticsearch | `9200` | Project Searching & Filtering. |
| **Broker** | RabbitMQ | `5672` | Event Bus (Async Communication). |
| **Storage** | Cloudinary | - | Media Storage (Direct Upload from Client). |

## 3. Critical Database Rules (The "Must-Follow")
1.  **UUID Strategy:**
    * SEMUA Primary Key (`id`) dan Foreign Key (`user_id`, `project_id`) **WAJIB UUID**.
    * DILARANG menggunakan Integer/Auto-increment.
2.  **Split Database (Isolation):**
    * **DB A (User):** Tables `users`, `user_credentials`, `refresh_tokens`, `follows`, `notifications`.
    * **DB B (Content):** Tables `projects`, `project_media`, `categories`, `comments`, `project_likes`, `comment_likes`, `bookmarks`, `collaborators`.
3.  **The "Ghost FK" Rule:**
    * Di **Content DB**, kolom `user_id` disimpan sebagai `UUID` biasa + `INDEX`.
    * **TIDAK BOLEH** ada constraint `REFERENCES users(id)` di Content DB karena tabel `users` tidak ada di sana.
4.  **Collaborators Table:**
    * Wajib memiliki kolom: `status` (ENUM: 'pending', 'accepted', 'rejected') dan `expired_at` (Timestamp).
    * Logic: Row dibuat saat invite, user harus accept via UI. Jika melewati `expired_at`, invitation dianggap kadaluarsa.

## 4. API Routing & Terminology
**Global Prefix:** `/v1`

### A. Auth Strategy (Cookie-Based)
* **Mechanism:** JWT disimpan dalam **HttpOnly Cookie**.
* **Login/Refresh Response:** Token **TIDAK** dikembalikan di JSON Body, melainkan via Header `Set-Cookie`.
* **Security:** Backend harus validasi `Cookie` header pada setiap request protected.

### B. Route Definitions (By Service)

#### 🔐 User Service (Express) -> Prefix `/v1`
* `POST /auth/register`, `/auth/login`, `/auth/refresh`
* `POST /auth/logout` (Clear Cookie & Revoke Token)
* `GET /users/me`, `PATCH /users/me` (Private Profile)
* `GET /users/:username` (Public Profile)
* `POST /users/:id/follow` (Toggle Follow)
* `GET /notifications`, `GET /notifications/count`
* `PATCH /notifications/read` (**Batch Read: Array User ID**)

#### 🎨 Content Service (Go) -> Prefix `/v1`
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
    * `GET /search` (Explore via Elastic).

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
    * `project.created` -> Worker index ke Elastic + Fan-out ke Redis followers.
    * `collaborator.invited` -> Worker create Notification.
    * `collaborator.responded` -> Worker create Notification to Owner.

### D. Pagination Strategy (Cursor Based)
* **Standard:** Semua endpoint `List` HARUS menggunakan **Cursor Pagination** untuk performa & konsistensi real-time.
* **Request:** `?cursor=eyJ...&limit=10`.
* **Response:**
    ```json
    "meta": {
      "curr_cursor": "...",
      "next_cursor": "...",
      "has_next": true,
      "limit": 10
    }
    ```
* **Offset/Page** (`?page=1`) **DILARANG** digunakan (Deprecated).

## 6. Development Guidelines
* **JSON Case:** Gunakan `snake_case` untuk response (`user_id`, `expired_at`).
* **Validation:** Validasi input ketat di Backend sebelum masuk logic.
* **Documentation:** Update `API_CONTRACT.md` sebelum coding fitur baru.

---

## 7. Collaborator Rules (Edge Cases)

### A. Leave & Ownership Transfer
* Collaborator bisa **leave sendiri** dari project.
* Owner **TIDAK BISA** kick/remove collaborator lain.
* Jika owner leave dan ada collaborator lain:
    * Ownership **berpindah** ke collaborator terbaru (yang paling akhir join).
* Jika owner adalah **satu-satunya pemilik** (Sole Owner):
    * Owner **TIDAK BISA leave**.
    * Diarahkan untuk **Archive** atau **Delete** showcase.

### B. Archive Feature
* Archive **HANYA tersedia** untuk Sole Owner.
* Showcase yang di-archive tidak muncul di feed/search, tapi masih bisa diakses via direct link oleh owner.

---

## 8. Like/Unlike Flow (Event-Driven)

### A. Frontend (Optimistic UI)
1. User klik tombol Like/Unlike.
2. UI **langsung berubah** (icon + counter) tanpa menunggu API.
3. Request `POST /v1/showcases/:id/like` dikirim di background.
4. Jika API error → **Rollback** ke state sebelumnya + tampilkan Toast error.

### B. Backend API (Content Service)
1. Cek tabel `project_likes`:
    * Belum ada → `INSERT` (Like).
    * Sudah ada → `DELETE` (Unlike/Toggle).
2. Gunakan **Unique Constraint** `(user_id, project_id)` untuk mencegah double-insert.
3. Publish event ke RabbitMQ:
    * Exchange: `talas.events`
    * Routing Key: `project.liked` atau `project.unliked`
    * Payload: `{ project_id, user_id, author_id, timestamp }`
4. Response: `{ "is_liked": true/false }` (tanpa count, agar < 50ms).

### C. Worker (Async Side Effects)
1. **Update Counter:** `UPDATE projects SET likes_count = likes_count +/- 1`.
2. **Update Redis Trending:** (Lihat Section 9).
3. **Sync Elasticsearch:** Update field `likes_count` di dokumen project.
4. **Send Notification:** Ke Author + semua Collaborators yang `ACCEPTED`.

---

## 9. Trending Feed (Rolling 7 Days Window)

### Strategy: Daily Buckets + ZUNIONSTORE

### A. Write Flow (Worker - saat project.liked)
```
ZINCRBY trending:{YYYY-MM-DD} 1 {project_id}
EXPIRE trending:{YYYY-MM-DD} 691200  # 8 hari dalam detik
```

### B. Read Flow (Content Service - GET /trending)
1. Generate 7 key names: `trending:H-0` sampai `trending:H-6`.
2. `ZUNIONSTORE trending:temp:{session_id} 7 key1 key2 ... key7`.
3. `ZREVRANGE trending:temp:{session_id} 0 N` untuk ambil Top N.
4. `EXPIRE trending:temp:{session_id} 10` atau langsung `DEL`.

**Hasil:** Efek "Trending Minggu Ini" yang rolling, bukan reset paksa tiap Senin.

---

## 11. Comment System Strategy (Flattened Replies)

### Logic:
* **Max Visual Nesting:** 3 Level (0, 1, 2).
* **Deep Replies (> Level 3):**
    * Render sejajar dengan parent terakhir (Flat).
    * Response API menyertakan field `reply_to: { username }`.
    * UI menampilkan: "**@username** [isi komentar]".

---

## 10. Notification Batching (Smart Aggregation)

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