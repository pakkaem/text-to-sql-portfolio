from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import os
import httpx
import uvicorn
import json
import time
from dotenv import load_dotenv

# Load .env file dari direktori yang sama dengan app.py
load_dotenv(os.path.join(os.path.dirname(os.path.abspath(__file__)), ".env"))

# Import torch/transformers hanya jika diperlukan (hemat waktu startup untuk API mode)
torch = None
AutoModelForCausalLM = None
AutoTokenizer = None

def _ensure_torch():
    global torch, AutoModelForCausalLM, AutoTokenizer
    if torch is None:
        import torch as _torch
        from transformers import AutoModelForCausalLM as _AM, AutoTokenizer as _AT
        torch = _torch
        AutoModelForCausalLM = _AM
        AutoTokenizer = _AT

app = FastAPI(
    title="Multi-Domain Text-to-SQL API",
    description="API AI untuk mengubah pertanyaan natural menjadi query SQL — mendukung multiple domain (HRIS, Smart City, dll)."
)

# --- SKEMA REQUEST & RESPONSE ---
class QueryRequest(BaseModel):
    schema_context: str
    question: str
    prev_sql: str = None      # Optional: SQL sebelumnya yang gagal (untuk retry)
    error_msg: str = None     # Optional: Error message dari eksekusi sebelumnya
    domain: str = "hris"      # Optional: domain database (hris, smartcity, dll)

class QueryResponse(BaseModel):
    sql_query: str

# Variabel global untuk model
model = None
tokenizer = None
model_mode = None  # "lora", "zero-shot", atau "groq"

# Groq API config
GROQ_API_URL = "https://api.groq.com/openai/v1/chat/completions"

# --- HELPER: Deteksi device ---
def get_device():
    if torch.cuda.is_available():
        return "cuda"
    return "cpu"

# --- LOAD: Mode LoRA (TinyLlama + Fine-tuned adapter) ---
def load_lora_model():
    global model, tokenizer
    from peft import PeftModel

    base_model_id = os.getenv("LORA_BASE_MODEL", "TinyLlama/TinyLlama-1.1B-Chat-v1.0")
    adapter_path = os.getenv("LORA_ADAPTER_PATH", "./checkpoint-1950")

    print(f"[LoRA Mode] Memuat base model: {base_model_id}")
    print(f"[LoRA Mode] Memuat adapter dari: {adapter_path}")

    tokenizer = AutoTokenizer.from_pretrained(base_model_id)
    device = get_device()
    print(f"[LoRA Mode] Device: {device}")

    base_model = AutoModelForCausalLM.from_pretrained(base_model_id).to(device)
    model = PeftModel.from_pretrained(base_model, adapter_path)
    model.eval()

# --- LOAD: Mode Zero-Shot (Model besar, tanpa fine-tuning) ---
def load_zeroshot_model():
    global model, tokenizer

    # Default: Qwen2.5-1.5B-Instruct — kecil, cepat di CPU (~3GB RAM), instruct model
    # Alternatif: "Qwen/Qwen2.5-3B-Instruct" (lebih bagus, ~6GB RAM)
    #            "microsoft/phi-2" (2.7B, base model, perlu format khusus)
    default_model = os.getenv("ZEROSHOT_MODEL_DEFAULT", "Qwen/Qwen2.5-1.5B-Instruct")
    model_id = os.getenv("ZEROSHOT_MODEL", default_model)

    print(f"[Zero-Shot Mode] Memuat model: {model_id}")

    tokenizer = AutoTokenizer.from_pretrained(model_id)
    device = get_device()
    print(f"[Zero-Shot Mode] Device: {device}")

    model = AutoModelForCausalLM.from_pretrained(
        model_id,
        torch_dtype=torch.float16 if device == "cuda" else torch.float32,
    ).to(device)
    model.eval()

# --- LOAD: Mode Groq (API Cloud, tanpa model lokal) ---
def call_groq_api(messages: list) -> str:
    """Mengirim request ke Groq API dan mengembalikan response.
    
    Dilengkapi retry logic dengan exponential backoff untuk menangani
    rate limit (HTTP 429) dari Groq API.
    """
    api_key = os.getenv("GROQ_API_KEY")
    model_name = os.getenv("GROQ_MODEL", "llama-3.1-8b-instant")

    if not api_key:
        raise Exception("GROQ_API_KEY tidak ditemukan di environment variables")

    headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
    }

    payload = {
        "model": model_name,
        "messages": messages,
        "temperature": 0.1,
        "max_tokens": 500,
        "top_p": 1,
    }

    max_retries = 3
    for attempt in range(max_retries):
        resp = httpx.post(GROQ_API_URL, json=payload, headers=headers, timeout=30)

        if resp.status_code == 429:
            # Rate limit hit — extract retry-after or use exponential backoff
            retry_after_header = resp.headers.get("retry-after")
            if retry_after_header:
                wait_time = float(retry_after_header)
            else:
                wait_time = 2 ** attempt  # 1s, 2s, 4s
            
            print(f"[Groq Rate Limit] Attempt {attempt + 1}/{max_retries} - Waiting {wait_time:.1f}s before retry...")
            
            if attempt < max_retries - 1:
                time.sleep(wait_time)
                continue
            else:
                # Last attempt — raise error
                raise Exception(
                    f"Groq API rate limit exceeded after {max_retries} retries. "
                    f"Last status: {resp.status_code}, body: {resp.text[:200]}"
                )

        # Non-429 errors — raise immediately
        resp.raise_for_status()

        data = resp.json()
        return data["choices"][0]["message"]["content"].strip()

    raise Exception(f"Groq API failed after {max_retries} retries")


# --- EVENT STARTUP: MEMUAT MODEL ---
@app.on_event("startup")
def load_ai_model():
    global model_mode

    # Baca mode dari environment variable: "groq", "lora", atau "zero-shot"
    model_mode = os.getenv("AI_MODEL_MODE", "groq").lower()

    print("=" * 60)
    print(f"AI Model Mode: {model_mode}")
    print("=" * 60)

    try:
        if model_mode == "groq":
            print("[Groq Mode] Menggunakan Groq API (tanpa model lokal)")
            print(f"[Groq Model] {os.getenv('GROQ_MODEL', 'llama-3.1-8b-instant')}")
            print(f"[Groq API Key] {'*' * 10}...{os.getenv('GROQ_API_KEY', '')[-4:]}")
            print("Siap menerima request!")
        elif model_mode == "lora":
            load_lora_model()
            print("Model berhasil dimuat! API siap menerima request.")
        else:
            load_zeroshot_model()
            print("Model berhasil dimuat! API siap menerima request.")
    except Exception as e:
        import traceback
        traceback.print_exc()
        print(f"Error memuat model: {e}")


# --- PROMPT BUILDING ---
def build_prompt(request: QueryRequest) -> str:
    """Membangun prompt yang sesuai dengan mode model."""

    rules = """### Rules:
- Hanya buat SELECT statement (tidak boleh INSERT/UPDATE/DELETE/DROP)
- Hanya JOIN tabel yang benar-benar DIBUTUHKAN untuk menjawab pertanyaan
- Pastikan kondisi JOIN menggunakan kolom yang SALING BERKORESPONDEN (misal: employee_id dengan id, bukan id dengan log_date)
- Gunakan HAVING untuk filter hasil aggregate (COUNT, SUM, AVG), bukan WHERE
- Hanya gunakan nama kolom yang ADA di schema"""

    contoh = "### " + _get_domain_examples(request.domain)

    # --- Mode Groq / Zero-Shot pakai format chat ---
    if model_mode in ("groq", "zero-shot"):
        # Retry prompt
        if request.prev_sql and request.error_msg:
            user_message = f"""Buatkan query SQL untuk database berikut.

{rules}

### Schema:
{request.schema_context}

### Pertanyaan:
{request.question}

### SQL Sebelumnya (Error):
{request.prev_sql}

### Error:
{request.error_msg}

Berikan HANYA query SQL yang sudah diperbaiki, tanpa penjelasan."""

            messages = [
                {"role": "system", "content": "Kamu adalah asisten database SQL. Hanya outputkan query SQL, tanpa markdown atau penjelasan."},
                {"role": "user", "content": user_message},
            ]
        else:
            user_message = f"""Buatkan query SQL untuk database berikut.

{rules}

{contoh}

### Schema:
{request.schema_context}

### Pertanyaan:
{request.question}

Berikan HANYA query SQL, tanpa penjelasan."""

            messages = [
                {"role": "system", "content": "Kamu adalah asisten database SQL. Hanya outputkan query SQL, tanpa markdown atau penjelasan."},
                {"role": "user", "content": user_message},
            ]

        # Gunakan chat template jika tokenizer mendukung
        if hasattr(tokenizer, "apply_chat_template"):
            prompt = tokenizer.apply_chat_template(messages, tokenize=False, add_generation_prompt=True)
        else:
            # Fallback: format manual untuk CodeLlama Instruct
            prompt = f"<s>[INST] <<SYS>>\n{messages[0]['content']}\n<</SYS>>\n\n{messages[1]['content']} [/INST]\n"

        return prompt

    # --- Mode LoRA (TinyLlama + Fine-tuned) pakai format ### Instruction ---
    else:
        if request.prev_sql and request.error_msg:
            return f"""### Instruction:
Kamu adalah asisten database AI. Query SQL sebelumnya gagal dieksekusi. Perbaiki query tersebut.

{rules}

### Schema:
{request.schema_context}

### Pertanyaan:
{request.question}

### SQL Sebelumnya (Error):
{request.prev_sql}

### Error:
{request.error_msg}

### Corrected SQL Query:
"""
        else:
            return f"""### Instruction:
Kamu adalah asisten database AI. Buat query SQL yang tepat untuk menjawab pertanyaan.

{rules}

{contoh}

### Schema:
{request.schema_context}

### Pertanyaan:
{request.question}

### SQL Query:
"""


def extract_sql(full_output: str) -> str:
    """Ekstrak query SQL dari output model."""
    if model_mode in ("groq", "zero-shot"):
        # CodeLlama Instruct: output setelah [/INST] atau setelah generation prompt
        # Bersihkan markdown code blocks jika ada
        output = full_output.strip()
        # Hapus ```sql ... ``` wrapper
        if "```sql" in output:
            output = output.split("```sql")[-1]
            output = output.split("```")[0].strip()
        elif "```" in output:
            output = output.split("```")[1]
            output = output.split("```")[0].strip()
        # Ambil baris pertama yang terlihat seperti SQL
        lines = output.strip().split("\n")
        sql_lines = []
        for line in lines:
            line = line.strip()
            if not line:
                continue
            upper = line.upper()
            if upper.startswith("SELECT") or upper.startswith("WITH") or \
               sql_lines or line.startswith("--"):
                sql_lines.append(line)
            elif "SELECT " in upper:
                sql_lines.append(line)
        return " ".join(sql_lines) if sql_lines else output.strip()
    else:
        # LoRA mode: format asli
        if "### Corrected SQL Query:" in full_output:
            return full_output.split("### Corrected SQL Query:")[-1].strip()
        else:
            return full_output.split("### SQL Query:")[-1].strip()


# --- ENDPOINT UTAMA ---
@app.post("/generate-sql", response_model=QueryResponse)
async def generate_sql(request: QueryRequest):
    try:
        if model_mode == "groq":
            # --- Groq API Mode (tanpa model lokal) ---
            messages = _build_groq_messages(request)
            raw_response = call_groq_api(messages)
            sql_result = extract_sql(raw_response)
            return QueryResponse(sql_query=sql_result)
        else:
            # --- Local Model Mode (LoRA / Zero-Shot) ---
            if not model or not tokenizer:
                raise HTTPException(status_code=500, detail="Model belum siap diload.")

            prompt = build_prompt(request)
            inputs = tokenizer(prompt, return_tensors="pt").to(model.device)

            with torch.no_grad():
                outputs = model.generate(
                    **inputs,
                    max_new_tokens=200,
                    temperature=0.1,
                    do_sample=True,
                    repetition_penalty=1.2,
                    pad_token_id=tokenizer.eos_token_id
                )

            new_tokens = outputs[0][inputs["input_ids"].shape[1]:]
            full_output = tokenizer.decode(new_tokens, skip_special_tokens=True)
            sql_result = extract_sql(full_output)
            return QueryResponse(sql_query=sql_result)

    except httpx.HTTPStatusError as e:
        raise HTTPException(status_code=502, detail=f"Groq API error: {e.response.status_code} - {e.response.text}")
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Gagal memproses request: {str(e)}")


def _get_domain_examples(domain: str) -> str:
    """Mengembalikan contoh pertanyaan-SQL yang sesuai dengan domain."""
    if domain == "smartcity":
        return """Contoh:
Pertanyaan: berapa total pelanggaran per jenis di setiap kota?
SQL: SELECT d.name, v.violation_type, COUNT(*) as total FROM violations v JOIN camera_locations cl ON v.camera_id = cl.id JOIN roads r ON cl.road_id = r.id JOIN districts d ON r.district_id = d.id GROUP BY d.name, v.violation_type ORDER BY d.name, total DESC

Pertanyaan: kamera mana yang paling banyak mendeteksi pelanggaran?
SQL: SELECT cl.location_desc, cl.camera_type, COUNT(*) as total_violations FROM violations v JOIN camera_locations cl ON v.camera_id = cl.id GROUP BY cl.id ORDER BY total_violations DESC LIMIT 5"""
    else:
        return """Contoh:
Pertanyaan: siapa saja yang bekerja di departemen Engineering?
SQL: SELECT e.name FROM employees e JOIN departments d ON e.department_id = d.id WHERE d.name = 'Engineering'

Pertanyaan: siapa yang mengerjakan lebih dari 1 proyek?
SQL: SELECT e.name, COUNT(ep.project_id) FROM employees e JOIN employee_projects ep ON e.id = ep.employee_id GROUP BY e.id HAVING COUNT(ep.project_id) > 1"""


def _build_groq_messages(request: QueryRequest) -> list:
    """Membangun messages array untuk Groq API."""
    rules = """Rules:
- Hanya buat SELECT statement (tidak boleh INSERT/UPDATE/DELETE/DROP)
- Hanya JOIN tabel yang benar-benar DIBUTUHKAN untuk menjawab pertanyaan
- Pastikan kondisi JOIN menggunakan kolom yang SALING BERKORESPONDEN (misal: employee_id dengan id, bukan id dengan log_date)
- Gunakan HAVING untuk filter hasil aggregate (COUNT, SUM, AVG), bukan WHERE
- Hanya gunakan nama kolom yang ADA di schema"""

    contoh = _get_domain_examples(request.domain)

    system_msg = "Kamu adalah asisten database SQL. HANYA outputkan query SQL, tanpa markdown, tanpa penjelasan apapun. Output harus berupa SQL query yang valid."

    if request.prev_sql and request.error_msg:
        user_msg = f"""Buatkan query SQL untuk database berikut.

{rules}

Schema:
{request.schema_context}

Pertanyaan: {request.question}

SQL Sebelumnya (Error):
{request.prev_sql}

Error: {request.error_msg}

Berikan HANYA query SQL yang sudah diperbaiki."""
    else:
        user_msg = f"""Buatkan query SQL untuk database berikut.

{rules}

{contoh}

Schema:
{request.schema_context}

Pertanyaan: {request.question}

Berikan HANYA query SQL."""

    return [
        {"role": "system", "content": system_msg},
        {"role": "user", "content": user_msg},
    ]

# --- ENDPOINT: EXPLAIN SQL ---
class ExplainRequest(BaseModel):
    sql: str
    question: str = ""
    domain: str = "hris"

class ExplainResponse(BaseModel):
    explanation: str

@app.post("/explain-sql", response_model=ExplainResponse)
async def explain_sql(request: ExplainRequest):
    """Menjelaskan SQL query dalam bahasa natural (Bahasa Indonesia)."""
    try:
        system_msg = "Kamu adalah guru database. Jelaskan query SQL berikut dengan bahasa yang mudah dipahami oleh non-programmer, dalam Bahasa Indonesia. Jelaskan apa yang dilakukan query, tabel apa saja yang dipakai, dan apa hasilnya. Maksimal 3-4 kalimat."

        user_msg = f"""Jelaskan query SQL ini dalam Bahasa Indonesia:

```sql
{request.sql}
```

{"Pertanyaan asli: " + request.question if request.question else ""}

Jelaskan dengan singkat dan jelas."""

        if model_mode == "groq":
            messages = [
                {"role": "system", "content": system_msg},
                {"role": "user", "content": user_msg},
            ]
            explanation = call_groq_api(messages)
            return ExplainResponse(explanation=explanation)
        else:
            # Untuk mode lokal, gunakan model yang sama
            if not model or not tokenizer:
                raise HTTPException(status_code=500, detail="Model belum siap diload.")

            messages = [
                {"role": "system", "content": system_msg},
                {"role": "user", "content": user_msg},
            ]

            if hasattr(tokenizer, "apply_chat_template"):
                prompt = tokenizer.apply_chat_template(messages, tokenize=False, add_generation_prompt=True)
            else:
                prompt = f"<s>[INST] <<SYS>>\n{system_msg}\n<</SYS>>\n\n{user_msg} [/INST]\n"

            inputs = tokenizer(prompt, return_tensors="pt").to(model.device)
            with torch.no_grad():
                outputs = model.generate(
                    **inputs,
                    max_new_tokens=300,
                    temperature=0.3,
                    do_sample=True,
                    pad_token_id=tokenizer.eos_token_id
                )
            new_tokens = outputs[0][inputs["input_ids"].shape[1]:]
            explanation = tokenizer.decode(new_tokens, skip_special_tokens=True).strip()
            return ExplainResponse(explanation=explanation)

    except httpx.HTTPStatusError as e:
        raise HTTPException(status_code=502, detail=f"Groq API error: {e.response.status_code}")
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Gagal menjelaskan SQL: {str(e)}")


# --- ENDPOINT: GENERATE INSIGHT ---
class InsightRequest(BaseModel):
    question: str
    sql_query: str
    data: list  # List of result rows (dicts)
    domain: str = "hris"

class InsightResponse(BaseModel):
    insight_summary: str
    business_explanation: str
    top_findings: list[str]

@app.post("/generate-insight", response_model=InsightResponse)
async def generate_insight(request: InsightRequest):
    """Generate AI-powered business insights from query results."""
    try:
        # Limit data to first 20 rows to keep prompt manageable
        data_sample = request.data[:20]
        data_json = json.dumps(data_sample, indent=2, default=str)

        # Count total rows for context
        total_rows = len(request.data)

        domain_context = ""
        if request.domain == "hris":
            domain_context = "Domain ini adalah HRIS (Human Resource Information System) — data karyawan, departemen, payroll, kehadiran, dan proyek."
        elif request.domain == "smartcity":
            domain_context = "Domain ini adalah Smart City — data pelanggaran lalu lintas, kamera, jalan, dan wilayah kota."

        system_msg = """Kamu adalah AI Business Analyst. Tugas kamu adalah menganalisis hasil query SQL dan memberikan insight bisnis yang bermakna dalam Bahasa Indonesia.

Kamu HARUS mengembalikan response dalam format JSON yang VALID (tanpa markdown code block):
{
  "insight_summary": "Satu kalimat insight utama yang impactful dan actionable",
  "business_explanation": "Penjelasan 2-3 kalimat tentang apa arti data ini dari sudut pandang bisnis, ditujukan untuk manajer/non-teknis",
  "top_findings": ["Finding 1", "Finding 2", "Finding 3"]
}

Rules:
- insight_summary: Harus singkat, impactful, dan langsung ke poin (maksimal 1 kalimat)
- business_explanation: Bahasa yang mudah dipahami non-teknis, menjelaskan implikasi bisnis
- top_findings: Maksimal 3 temuan utama, urutkan dari yang paling penting
- Jika data kosong, jelaskan bahwa tidak ada data yang ditemukan
- Selalu dalam Bahasa Indonesia
- JANGAN gunakan markdown atau code block di response"""

        user_msg = f"""Analisis hasil query berikut:

Pertanyaan User: {request.question}

SQL Query:
```sql
{request.sql_query}
```

{domain_context}

Hasil Data ({total_rows} baris total, showing {len(data_sample)} sample):
```json
{data_json}
```

Berikan analisis bisnis dalam format JSON."""

        if model_mode == "groq":
            messages = [
                {"role": "system", "content": system_msg},
                {"role": "user", "content": user_msg},
            ]
            raw_response = call_groq_api(messages)

            # Parse JSON from response
            insight_data = _parse_json_response(raw_response)
            return InsightResponse(
                insight_summary=insight_data.get("insight_summary", "Tidak dapat menghasilkan insight dari data ini."),
                business_explanation=insight_data.get("business_explanation", "Data tidak cukup untuk analisis bisnis."),
                top_findings=insight_data.get("top_findings", ["Tidak ada temuan yang dapat diidentifikasi."]),
            )
        else:
            # Local model mode
            if not model or not tokenizer:
                raise HTTPException(status_code=500, detail="Model belum siap diload.")

            messages = [
                {"role": "system", "content": system_msg},
                {"role": "user", "content": user_msg},
            ]

            if hasattr(tokenizer, "apply_chat_template"):
                prompt = tokenizer.apply_chat_template(messages, tokenize=False, add_generation_prompt=True)
            else:
                prompt = f"<s>[INST] <<SYS>>\n{system_msg}\n<</SYS>>\n\n{user_msg} [/INST]\n"

            inputs = tokenizer(prompt, return_tensors="pt").to(model.device)
            with torch.no_grad():
                outputs = model.generate(
                    **inputs,
                    max_new_tokens=500,
                    temperature=0.3,
                    do_sample=True,
                    pad_token_id=tokenizer.eos_token_id
                )
            new_tokens = outputs[0][inputs["input_ids"].shape[1]:]
            raw_response = tokenizer.decode(new_tokens, skip_special_tokens=True).strip()

            insight_data = _parse_json_response(raw_response)
            return InsightResponse(
                insight_summary=insight_data.get("insight_summary", "Tidak dapat menghasilkan insight dari data ini."),
                business_explanation=insight_data.get("business_explanation", "Data tidak cukup untuk analisis bisnis."),
                top_findings=insight_data.get("top_findings", ["Tidak ada temuan yang dapat diidentifikasi."]),
            )

    except httpx.HTTPStatusError as e:
        raise HTTPException(status_code=502, detail=f"Groq API error: {e.response.status_code}")
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Gagal menghasilkan insight: {str(e)}")


def _parse_json_response(raw: str) -> dict:
    """Parse JSON from LLM response, handling markdown code blocks."""
    text = raw.strip()
    # Remove markdown code block if present
    if "```json" in text:
        text = text.split("```json", 1)[1]
        text = text.split("```", 1)[0].strip()
    elif "```" in text:
        text = text.split("```", 1)[1]
        text = text.split("```", 1)[0].strip()
    
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        # Try to find JSON object in the text
        start = text.find("{")
        end = text.rfind("}") + 1
        if start != -1 and end > start:
            try:
                return json.loads(text[start:end])
            except json.JSONDecodeError:
                pass
        # Fallback
        return {
            "insight_summary": text[:200] if text else "Tidak dapat menghasilkan insight.",
            "business_explanation": "Terjadi kesalahan saat memproses analisis bisnis.",
            "top_findings": ["Data tidak dapat dianalisis secara otomatis."],
        }


if __name__ == "__main__":
    uvicorn.run(app, host="127.0.0.1", port=8000)
