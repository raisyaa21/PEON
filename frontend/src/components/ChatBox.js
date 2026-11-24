"use client";
import { useCallback, useEffect, useRef, useState } from "react";
import { api, fmt } from "@/lib/api";

// role: "mahasiswa" | "staf"; studentId wajib untuk staf
export default function ChatBox({ role, studentId }) {
  const [msgs, setMsgs] = useState([]);
  const [text, setText] = useState("");
  const [err, setErr] = useState("");
  const endRef = useRef(null);
  const qs = role === "staf" ? `?student_id=${studentId}` : "";

  const load = useCallback(
    () => api(`/api/messages${qs}`).then(setMsgs).catch((e) => setErr(e.message)),
    [qs]
  );

  useEffect(() => {
    load();
    const t = setInterval(load, 5000);
    return () => clearInterval(t);
  }, [load]);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [msgs.length]);

  async function send(e) {
    e.preventDefault();
    if (!text.trim()) return;
    try {
      await api("/api/messages", { method: "POST", body: { body: text, student_id: studentId } });
      setText("");
      setErr("");
      load();
    } catch {
      setErr("Pesan gagal terkirim. Ketuk Kirim untuk mencoba lagi.");
    }
  }

  return (
    <>
      <div className="chat-log">
        {msgs.length === 0 && <div className="muted">Belum ada pesan. Mulai percakapan di bawah.</div>}
        {msgs.map((m) => (
          <div key={m.id} className={`bubble ${m.sender_role === role ? "me" : ""}`}>
            <small>
              {m.sender_name} · {fmt(m.created_at)}
            </small>
            {m.body}
          </div>
        ))}
        <div ref={endRef} />
      </div>
      {err && <div className="chat-err">⚠ {err}</div>}
      <form className="chat-form" onSubmit={send}>
        <input
          className="input"
          placeholder="Ketik pesan pertanyaan atau…"
          value={text}
          onChange={(e) => setText(e.target.value)}
          maxLength={2000}
        />
        <button className="btn btn-primary">➤</button>
      </form>
    </>
  );
}