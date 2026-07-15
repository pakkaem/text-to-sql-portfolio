# Environment Variables Reference

A listing of all configuration variables parsed by services at runtime, categorized by service and security level.

---

## 1. Frontend Service (`frontend-service`)

These variables configure the SvelteKit Node.js server and client bundle.

| Variable | Required | Default | Security Level | Description |
| :--- | :---: | :---: | :---: | :--- |
| `PORT` | No | `3000` | **Low** | The port the SvelteKit node application binds to. |
| `PUBLIC_API_URL` | **Yes** | `http://localhost:8080` | **Medium** | The client-facing URL of the backend API. Must start with `PUBLIC_` to be readable in Svelte client code. |

---

## 2. Backend Service (`backend-service`)

These variables configure the Go (Gin) API server.

| Variable | Required | Default | Security Level | Description |
| :--- | :---: | :---: | :---: | :--- |
| `PORT` / `BACKEND_PORT` | No | `8080` | **Low** | The port the Gin HTTP server binds to. `PORT` takes precedence. |
| `AI_SERVICE_URL` | **Yes** | `http://127.0.0.1:8000` | **Medium** | Internal URL of the FastAPI AI Service container. |
| `CORS_ORIGINS` | No | `*` | **Medium** | List of allowed CORS origins (comma-separated). Set to frontend domain in production. |
| `HRIS_DB_PATH` / `DB_NAME`| No | `./hris.db` | **Low** | File path to the HRIS SQLite database. |
| `SMARTCITY_DB_PATH` | No | `./smartcity.db` | **Low** | File path to the Smart City SQLite database. |
| `GIN_MODE` | No | `debug` | **Low** | Set to `release` in production to disable verbose debugging logs. |

---

## 3. AI Service (`ai-service`)

These variables configure the FastAPI AI Python service.

| Variable | Required | Default | Security Level | Description |
| :--- | :---: | :---: | :---: | :--- |
| `PORT` | No | `8000` | **Low** | The port uvicorn binds to. |
| `AI_MODEL_MODE` | No | `groq` | **Low** | Execution mode: `groq` (default, API Cloud), `zero-shot` (local), `lora` (local). |
| `GROQ_API_KEY` | **Yes** (in groq mode) | — | **Critical** | Groq Cloud platform API credential. **Never expose this value publicly.** |
| `GROQ_MODEL` | No | `llama-3.1-8b-instant` | **Low** | LLM model model-ID deployed on Groq. |
| `GROQ_TIMEOUT` | No | `30.0` | **Low** | Request timeout in seconds for Groq API posts. |
| `AI_SERVICE_HOST` | No | `0.0.0.0` | **Low** | Bind host when running Python app directly. |
| `AI_SERVICE_PORT` | No | `8000` | **Low** | Bind port when running Python app directly. |
