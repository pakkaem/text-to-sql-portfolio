# Production Security Checklist

A checklist of security standards that must be verified before releasing the application to the public internet.

---

## 1. Secrets Isolation
- [ ] **No Committed API Keys**: Verify that no `.env` files or API key credentials are saved inside Git history.
- [ ] **Docker Cache Security**: Verify that `GROQ_API_KEY` is passed to the container via runtime environment variables rather than hardcoded in the `Dockerfile` or build arguments (`ARG`).

## 2. CORS Policies
- [ ] **CORS Restricted**: Change `CORS_ORIGINS` in the backend service production config from `*` to the specific frontend domain (e.g. `https://frontend.up.railway.app`).
- [ ] **AllowCredentials Audit**: Confirm that CORS configurations correctly restrict credentials headers to authenticated clients only.

## 3. Database & SQL Execution Security
- [ ] **Read-Only Enforcement**: Verify that the backend checks `isReadOnlySQL()` on every generated SQL query. Only `SELECT` and `WITH` statements should be allowed to run.
- [ ] **Safe SQL Parsing**: Ensure that SQL syntax queries from the AI are stripped of markdown comments, extra spaces, and invalid drop/truncate sequences.
- [ ] **LIMIT Constraints**: Verify that `ensureLimit` is applied to all queries. Maximum output result sets should not exceed 100 rows to prevent Denial-of-Service (DoS) and Out-Of-Memory (OOM) failures.

## 4. Container Hardening
- [ ] **Non-Root Runtime Execution**: Ensure every `Dockerfile` contains `USER appuser` (UID 10001) to isolate container user privileges from the host namespace.
- [ ] **Read-Only Container Root FS** (Recommended): In hosting platforms, run containers with a read-only root filesystem (using volume mounts only for `/app/data` database writes) to mitigate runtime filesystem write attacks.

## 5. Debug Mode Disabling
- [ ] **Gin Production Mode**: Set the environment variable `GIN_MODE=release` on the backend service to disable trace dumps and debugging logs.
- [ ] **FastAPI Docs Control** (Optional): Set `docs_url=None` and `redoc_url=None` in `FastAPI(...)` constructor to disable API documentation endpoints inside the public AI Service if the AI container is ever exposed. (Currently isolated behind private networks, which is secure by default).
