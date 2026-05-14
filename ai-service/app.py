from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import torch
from transformers import AutoModelForCausalLM, AutoTokenizer
from peft import PeftModel
import uvicorn

app = FastAPI(
    title="HRIS Text-to-SQL API",
    description="API AI untuk mengubah pertanyaan natural menjadi query SQL."
)

# --- SKEMA REQUEST & RESPONSE ---
class QueryRequest(BaseModel):
    schema_context: str
    question: str

class QueryResponse(BaseModel):
    sql_query: str

# Variabel global untuk model
model = None
tokenizer = None

# --- EVENT STARTUP: MEMUAT MODEL ---
@app.on_event("startup")
def load_ai_model():
    global model, tokenizer
    print("Memuat base model dan LoRA adapter... (Mohon tunggu)")
    
    base_model_id = "TinyLlama/TinyLlama-1.1B-Chat-v1.0"
    adapter_path = "./checkpoint-200" 
    
    try:
        tokenizer = AutoTokenizer.from_pretrained(base_model_id)
        
        # Deteksi otomatis: pakai GPU jika ada, pakai CPU jika tidak ada
        device = "cuda" if torch.cuda.is_available() else "cpu"
        print(f"Berjalan menggunakan device: {device}")
        
        # LOAD BASE MODEL TANPA "device_map" DAN "torch_dtype"
        base_model = AutoModelForCausalLM.from_pretrained(
            base_model_id
        ).to(device) # Pindahkan langsung ke CPU/GPU secara eksplisit
        
        # Suntikkan otak LoRA
        model = PeftModel.from_pretrained(base_model, adapter_path)
        model.eval()
        
        print("Model berhasil dimuat! API siap menerima request.")
    except Exception as e:
        import traceback
        traceback.print_exc() # Menampilkan detail error yang lebih lengkap
        print(f"Error memuat model: {e}")

# --- ENDPOINT UTAMA ---
@app.post("/generate-sql", response_model=QueryResponse)
async def generate_sql(request: QueryRequest):
    if not model or not tokenizer:
        raise HTTPException(status_code=500, detail="Model belum siap diload.")

    # Format Prompt Template persis seperti di Colab
    prompt = f"""### Instruction:
Kamu adalah asisten database AI. Berdasarkan skema tabel berikut, buatlah query SQL yang tepat untuk menjawab pertanyaan.

### Schema:
{request.schema_context}

### Question:
{request.question}

### SQL Query:
"""
    
    inputs = tokenizer(prompt, return_tensors="pt").to(model.device)
    
    try:
        with torch.no_grad():
            outputs = model.generate(
                **inputs,
                max_new_tokens=50,
                temperature=0.1,
                do_sample=True,
                repetition_penalty=1.2,
                pad_token_id=tokenizer.eos_token_id
            )
            
        # Potong hasil agar hanya menyisakan query SQL-nya saja
        full_output = tokenizer.decode(outputs[0], skip_special_tokens=True)
        sql_result = full_output.split("### SQL Query:")[-1].strip()
        
        return QueryResponse(sql_query=sql_result)
        
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Gagal memproses request: {str(e)}")

if __name__ == "__main__":
    uvicorn.run(app, host="127.0.0.1", port=8000)