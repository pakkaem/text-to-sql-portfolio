# Release Checklist

Perform these checks in order before launching the deployment live for public use.

---

## Pre-Release Phase (Local)
- [ ] Run test builds for SvelteKit, Go, and Python services locally.
- [ ] Confirm git status is clean and all edits are committed.
- [ ] Verify root `.env` is omitted from Git indexing.

## Deployment Phase (Cloud Console)
- [ ] Deploy `ai-service` container:
  - [ ] Add `GROQ_API_KEY` env var.
  - [ ] Map port 8000 (private network).
- [ ] Deploy `backend-service` container:
  - [ ] Configure `AI_SERVICE_URL` target to reference the private AI service domain.
  - [ ] Create persistent volume mount mapped to `/app/data`.
  - [ ] Add custom `CORS_ORIGINS` to allow only the frontend's upcoming URL.
  - [ ] Set `GIN_MODE=release`.
- [ ] Deploy `frontend-service` container:
  - [ ] Set `PUBLIC_API_URL` env to point to the newly generated public URL of the Go backend.

## Post-Release Verification Phase (E2E)
- [ ] Load the frontend URL in the browser and confirm it serves content (Status code 200).
- [ ] Check health dots in the header:
  - [ ] Confirm both dots are green (indicating database and AI service connectivity checks succeed).
- [ ] Test E2E AI pipeline:
  - [ ] Select **HRIS** -> Ask suggested query: "Who earns the highest salary?" -> Verify SQL generates, runs, and renders correct rows.
  - [ ] Click **Explain SQL** -> Confirm business explanation renders correctly.
  - [ ] Export CSV -> Confirm file downloads with CSV formatting.
  - [ ] Select **Smart City** -> Ask suggested query: "Which camera detects the most violations?" -> Confirm SQL query runs and maps violation tables.
