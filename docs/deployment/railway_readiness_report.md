# Railway Readiness Report

An audit evaluating the platform services against Railway-specific deployment specifications and configurations.

---

## 1. Service Audits

### A. Frontend Service (`frontend-service`)
- **Port Mapping**: Satisfied. Railway automatically injects `PORT=3000` (or defaults to the exposed port) and routes external traffic to it.
- **Startup Command**: Satisfied. The container uses `CMD ["node", "build/index.js"]` which executes the SvelteKit Node runner built with `adapter-node`.
- **Health Check**: Satisfied. The custom `wget` health test in docker compose translates directly to Railway's automated HTTP health paths checking `/` with standard timeout limits.
- **Variables**: Satisfied. SvelteKit public dynamic env resolves `PUBLIC_API_URL` at runtime.

### B. Backend Service (`backend-service`)
- **Port Mapping**: Satisfied. Resolves port bindings via `getEnv("PORT", ...)` to support Railway's random port injection.
- **Startup Command**: Satisfied. Executes statically compiled Linux Go binary `/app/backend`.
- **Health Check**: Satisfied. GET `/health` is fully functional and performs dependency check pings on database connections and downstream AI service endpoints.
- **Persistence**: Satisfied. SQLite databases are mapped to `/app/data/`, ready for Railway persistent volume mount attachments.

### C. AI Service (`ai-service`)
- **Port Mapping**: Satisfied. Exposes port 8000. Start command resolves binding automatically using standard flags.
- **Startup Command**: Satisfied. Executes `uvicorn app:app --host 0.0.0.0 --port ${PORT}` resolving injected ports.
- **Health Check**: Satisfied. Serves a lightweight GET `/health` endpoint returning server status.
- **Resource Constraints**: Warned. Local model inference modes (`zero-shot` or `lora`) are disabled by default. If enabled in production, the container memory allocation must be scaled to at least 4GB RAM to prevent container crashes. For the default `groq` mode, 512MB RAM is fully sufficient.

---

## 2. Compatibility Mapping Table

| Service | Expose Port | Health Endpoint | Startup Command | Ephemeral Filesystem | Volume Required |
| :--- | :---: | :--- | :--- | :---: | :---: |
| **Frontend** | `3000` | `/` | `node build/index.js` | Yes | No |
| **Backend** | `8080` | `/health` | `/app/backend` | No (Volume mapped) | Yes (`/app/data`) |
| **AI Service**| `8000` | `/health` | `uvicorn app:app --host 0.0.0.0 --port ${PORT}` | Yes | No |

---

## 3. Verdict
The platform has **100% compatibility** with Railway container rules. There are no technical blockers preventing successful monorepo cloud deployment.
