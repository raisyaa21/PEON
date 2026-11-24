"use client";
import { useEffect, useState } from "react";
import { api, fmt } from "@/lib/api";
import ChatBox from "@/components/ChatBox";

export default function StafBantuan() {
  const [convos, setConvos] = useState([]);
  const [sel, setSel] = useState(null);

  useEffect(() => {
    const load = () => api("/api/staff/conversations").then(setConvos).catch(() => {});
    load();
    const t = setInterval(load, 8000);
    return () => clearInterval(t);
  }, []);

  return (
    <>
      <div className="page-head">
        <h1>Pesan Bantuan</h1>
        <p className="muted">Balas pertanyaan mahasiswa. Seluruh percakapan tersimpan sebagai log pelayanan.</p>
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "280px 1fr", gap: 16 }}>
        <div className="chat-shell">
          {convos.length === 0 && <p className="muted" style={{ padding: 14 }}>Belum ada percakapan.</p>}
          {convos.map((c) => (
            <div
              key={c.student_id}
              onClick={() => setSel(c)}
              style={{
                padding: 12,
                cursor: "pointer",
                borderBottom: "1px solid var(--border)",
                background: sel?.student_id === c.student_id ? "#eef2ff" : "transparent",
              }}
            >
              <b>{c.name}</b>
              <div className="muted">{c.identifier}</div>
              <div className="muted" style={{ whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
                {c.last_body}
              </div>
              <div className="muted" style={{ fontSize: ".68rem" }}>{fmt(c.last_at)}</div>
            </div>
          ))}
        </div>
        <div className="chat-shell">
          {sel ? (
            <>
              <div className="chat-head">
                <span className="avatar">{sel.name[0]}</span>
                <div><b>{sel.name}</b><small>{sel.identifier}</small></div>
              </div>
              <ChatBox key={sel.student_id} role="staf" studentId={sel.student_id} />
            </>
          ) : (
            <p className="muted" style={{ padding: 18 }}>Pilih percakapan di sebelah kiri.</p>
          )}
        </div>
      </div>
    </>
  );
}