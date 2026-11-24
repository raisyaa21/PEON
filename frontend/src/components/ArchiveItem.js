"use client";
import { useState } from "react";
import { downloadPath, fmtDay } from "@/lib/api";

export default function ArchiveItem({ item }) {
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  async function download() {
    setErr("");
    setBusy(true);
    try {
      const name = `${item.letter_type}-${item.ticket_no}.pdf`.replace(/[\s/]+/g, "_");
      await downloadPath(`/api/tickets/${item.ticket_id}/letter?kind=final`, name);
    } catch {
      setErr("Dokumen PDF tidak dapat diunduh saat ini.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="tcard" style={{ cursor: "default" }}>
      <b>{item.letter_type}</b>
      <div className="muted">
        No. {item.letter_no} · Memuat Tanda Tangan Elektronik (TTE) &amp; Kode QR Legalitas
      </div>
      <div className="row between" style={{ margin: "6px 0 10px" }}>
        <span className="muted">Diterbitkan: {fmtDay(item.issued_at)}</span>
        <span className="muted">PDF • {item.size_kb} KB</span>
      </div>
      <button className="btn btn-primary btn-block" onClick={download} disabled={busy}>
        {busy ? "Mengunduh…" : "⤓ Unduh PDF"}
      </button>
      {err && (
        <div className="alert alert-bad" style={{ margin: "10px 0 0" }}>
          <b>⚠ Gagal Unduh</b>
          <div>{err}</div>
        </div>
      )}
    </div>
  );
}