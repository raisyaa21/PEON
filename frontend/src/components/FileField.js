"use client";
import { useRef } from "react";
import { validateFile } from "@/lib/api";

// onPick(file | null, errorMessage), onView() opsional
export default function FileField({ label, file, error, onPick, onView }) {
  const ref = useRef(null);

  function handle(e) {
    const f = e.target.files[0] || null;
    if (!f) return onPick(null, "");
    const msg = validateFile(f);
    if (msg) e.target.value = "";
    onPick(msg ? null : f, msg);
  }

  return (
    <>
      <div className={`slot ${file ? "ok" : ""} ${error ? "bad" : ""}`}>
        <span>
          {file ? `${file.name} · ${(file.size / 1048576).toFixed(1)} MB` : label}
        </span>
        <span className="row">
          {file && onView && (
            <button type="button" className="link" onClick={onView}>
              Terunggah · Lihat pratinjau
            </button>
          )}
          <button type="button" className="link" onClick={() => ref.current.click()}>
            {file ? "Ganti" : "Seret berkas atau pilih file"}
          </button>
        </span>
        <input
          ref={ref}
          type="file"
          hidden
          accept=".pdf,.jpg,.jpeg,application/pdf,image/jpeg"
          onChange={handle}
        />
      </div>
      {error && <div className="slot-err">⚠ {error}</div>}
    </>
  );
}