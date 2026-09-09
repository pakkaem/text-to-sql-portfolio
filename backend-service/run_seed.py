"""Script untuk seed database HRIS. Jalankan dari folder backend-service."""
import sqlite3
import sys
import os

def run_seed(filename):
    if not os.path.exists(filename):
        print(f"ERROR: File '{filename}' tidak ditemukan!")
        return False
    
    conn = sqlite3.connect("hris.db")
    try:
        with open(filename, "r") as f:
            sql = f.read()
        conn.executescript(sql)
        print(f"SUCCESS: Data dari '{filename}' berhasil di-seed!")
        return True
    except Exception as e:
        print(f"ERROR: Gagal seed dari '{filename}': {e}")
        return False
    finally:
        conn.close()

if __name__ == "__main__":
    # Default: seed-data.sql, atau file yang diargumentasikan
    target = sys.argv[1] if len(sys.argv) > 1 else "seed-data.sql"
    run_seed(target)