# Cloud Deployment Guide — Monorepo Architecture

This guide describes how to deploy the Multi-Domain AI Analytics Platform to a containerized cloud environment (such as Railway or custom Docker hosts) in a monorepo setup.

---

## 1. Cloud Target Architecture

```
                       Internet (HTTPS)
                              │
                              ▼
                   [frontend-service] (SvelteKit)
                   (Public domain e.g., *.up.railway.app)
                              │
                    Client-side HTTP Fetch
                    (PUBLIC_API_URL target)
                              │
                              ▼
                    [backend-service] (Go/Gin)
                   (Public domain e.g., *.up.railway.app)
                              │
                    Internal Network Call
                    (AI_SERVICE_URL target)
                              │
                              ▼
                     [ai-service] (FastAPI)
                   (Internal Only, Port 8000)
                              │
                              ▼
                           Groq API
```

---

## 2. Monorepo Service Deployment on Railway

To deploy this monorepo to Railway, you will instantiate **three separate services** linked to the same GitHub repository:

### A. AI Service (`ai-service`)
1. **Source**: Link the repository.
2. **Root Directory**: Set to `/ai-service`.
3. **Build Settings**: Railway automatically detects the Python `Dockerfile` in the directory.
4. **Networking**: Do *not* generate a public domain (keep it private). Expose Port `8000`.
5. **Environment Variables**: Set `GROQ_API_KEY`, `AI_MODEL_MODE=groq`, `GROQ_MODEL=llama-3.1-8b-instant`.

### B. Backend Service (`backend-service`)
1. **Source**: Link the repository.
2. **Root Directory**: Set to `/backend-service`.
3. **Build Settings**: Railway automatically detects the Go `Dockerfile` in the directory.
4. **Networking**: Expose Port `8080` and **generate a public domain** (e.g. `https://backend-production.up.railway.app`).
5. **Volumes**: Create a **Railway Volume** mapped to mount path `/app/data` to persist the SQLite databases.
6. **Environment Variables**:
   - `PORT=8080`
   - `AI_SERVICE_URL=http://ai-service.railway.internal:8000` (pointing to the private AI service name via Railway private networking)
   - `HRIS_DB_PATH=/app/data/hris.db`
   - `SMARTCITY_DB_PATH=/app/data/smartcity.db`
   - `CORS_ORIGINS=https://frontend-production.up.railway.app` (restrict this to the frontend domain)
   - `GIN_MODE=release`

### C. Frontend Service (`frontend-service`)
1. **Source**: Link the repository.
2. **Root Directory**: Set to `/frontend-service`.
3. **Build Settings**: Railway detects SvelteKit's Node.js `Dockerfile`.
4. **Networking**: Expose Port `3000` and **generate a public domain** (e.g. `https://frontend-production.up.railway.app`).
5. **Environment Variables**:
   - `PORT=3000`
   - `PUBLIC_API_URL=https://backend-production.up.railway.app` (the public backend domain created above)

---

## 3. Database Strategy: SQLite Persistence

- **Volume Mount**: A persistent volume is mounted to `/app/data` in the backend service.
- **Initialization**: Go handles database creation. If `/app/data/hris.db` and `/app/data/smartcity.db` do not exist, they will be initialized and seeded on startup.
- **Ephemerality Protection**: If the volume is omitted, redeploying the backend service will reset database states to original seed templates, erasing query/schema telemetry.
