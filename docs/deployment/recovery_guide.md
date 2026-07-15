# Disaster Recovery & Operations Guide

This guide documents response protocols for typical operational issues in production.

---

## 1. Issue: SvelteKit UI shows "API Offline" (Backend Unreachable)
*   **Symptom**: Red error card in frontend or offline health indicators.
*   **Diagnostics**:
    1. Check public Go backend logs via the cloud platform (e.g. Railway deploy dashboard).
    2. Confirm if the backend crashed due to database lock or filesystem permissions:
       - Log: `Gagal membuat skema: read-only filesystem` or `permission denied`.
       - Fix: The backend needs write permissions to the `/app/data` volume folder to create/open SQLite database files on first run. Verify volume mounts and ownership.
    3. Verify frontend `PUBLIC_API_URL` environment variable matches the backend's current exposed HTTPS endpoint.

---

## 2. Issue: Backend returns 502 Bad Gateway ("Gagal terhubung ke AI Service")
*   **Symptom**: Chat queries fail with a 502 Bad Gateway card in the UI.
*   **Diagnostics**:
    1. Confirm the `ai-service` container is running and healthy.
    2. Check backend `AI_SERVICE_URL` variable. Ensure it targets the private DNS name of the AI service container (e.g., `http://ai-service.railway.internal:8000`), not `localhost`.
    3. Check AI Service logs for OOM (Out of Memory) crashes, which occur if `AI_MODEL_MODE` is incorrectly set to local model modes (`lora` / `zero-shot`) on a cloud instance with low memory (<4GB RAM). Default mode must be `groq` to run on lightweight containers.

---

## 3. Issue: Database Locked (`database is locked` SQLite error)
*   **Symptom**: SQL queries from users hang or fail with database write errors.
*   **Diagnostics**:
    - SQLite supports concurrent reads, but writes lock the database. Since our platform is read-only for users, writes only happen during initial database seeding on first deploy.
    - If a lock happens, restart the `backend-service` container to drop active connections and clear the file lock.

---

## 4. Issue: Groq API Rate Limit or Timeouts (HTTP 429 / 504 Gateway Timeout)
*   **Symptom**: Python AI service logs print `[Groq Rate Limit] Attempt X/3 - Waiting...` or requests terminate due to timeouts.
*   **Diagnostics**:
    1. Check python logs to confirm whether rate limit limits have been hit on the Groq API key.
    2. If timeouts occur, check if the timeout setting `GROQ_TIMEOUT` is too low for the current network conditions (increase to `45.0` or `60.0`).
    3. For high-volume production traffic, configure an API key rotation or upgrade Groq tiers to increase tokens per minute (TPM).
