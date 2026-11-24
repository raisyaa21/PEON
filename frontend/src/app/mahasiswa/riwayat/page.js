"use client";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import ArchiveItem from "@/components/ArchiveItem";

export default function Riwayat() {
  const [items, setItems] = useState(null);
  const [q, setQ] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    api("/api/archive").then(setItems).catch((e) => setError(e.message));
  }, []);

  const shown = (items || []).filter((x) =>
    `${x.letter_no} ${x.letter_type} ${x.ticket_no}`.toLowerCase().includes(q.toLowerCase())
  );

  return (
    <>
      <div className="page-head">
        <h1>Riwayat & Arsip Surat</h1>
      </div>
      <div className="field">
        <input
          className="input"
          style={{ background: "#fff", border: "1px solid var(--border)" }}
          placeholder="Cari nomor surat atau jenis permohonan…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>
      {error && <div className="alert alert-bad">{error}</div>}
      {!items ? (
        <p className="muted">Memuat…</p>
      ) : shown.length === 0 ? (
        <div className="card"><p className="muted">{items.length === 0 ? "Belum ada surat yang diarsipkan." : "Tidak ada surat yang cocok."}</p></div>
      ) : (
        shown.map((it) => <ArchiveItem key={it.ticket_id} item={it} />)
      )}
    </>
  );
}