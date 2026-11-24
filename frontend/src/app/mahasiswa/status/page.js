"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { api, fmt } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";
import StatusTracker from "@/components/StatusTracker";

function TicketCard({ t }) {
  const finished = t.status === "completed";
  const [open, setOpen] = useState(!finished);
  const [detail, setDetail] = useState(null);

  // ambil ulang detail setiap list ter-poll dan updated_at berubah
  useEffect(() => {
    if (!open) return;
    api(`/api/tickets/${t.id}`).then(setDetail).catch(() => {});
  }, [open, t.id, t.updated_at]);

  return (
    <div className="tcard" onClick={() => setOpen((v) => !v)}>
      <div className="row between">
        <div>
          <b>{t.letter_type}</b>
          <div className="muted">#{t.ticket_no} · diajukan {fmt(t.created_at)}</div>
        </div>
        <StatusBadge status={t.status} />
      </div>

      {open && detail && (
        <div onClick={(e) => e.stopPropagation()}>
          <StatusTracker ticket={detail} />
          {t.status === "revision" && (
            <>
              <div className="note-box"><b>Catatan staf:</b> {t.note}</div>
              <Link href={`/mahasiswa/revisi/${t.id}`} className="btn btn-primary btn-block">
                Revisi Dokumen
              </Link>
            </>
          )}
          {t.status === "rejected" && (
            <div className="alert alert-bad" style={{ marginTop: 12 }}><b>Ditolak.</b> {t.note}</div>
          )}
        </div>
      )}
    </div>
  );
}

export default function StatusPage() {
  const [items, setItems] = useState(null);
  const [error, setError] = useState("");

  // polling 5 detik -> real-time
  useEffect(() => {
    const load = () => api("/api/tickets").then(setItems).catch((e) => setError(e.message));
    load();
    const t = setInterval(load, 5000);
    return () => clearInterval(t);
  }, []);

  const processing = items?.filter((x) => !["completed", "rejected"].includes(x.status)).length ?? 0;
  const finished = items?.filter((x) => x.status === "completed").length ?? 0;

  return (
    <>
      <div className="page-head">
        <h1>Pelacakan Status</h1>
      </div>
      {error && <div className="alert alert-bad">{error}</div>}

      <div className="stats">
        <div className="stat"><small>Sedang Diproses</small><b>{processing}</b></div>
        <div className="stat"><small>Selesai</small><b>{finished}</b></div>
      </div>

      {!items ? (
        <p className="muted">Memuat…</p>
      ) : items.length === 0 ? (
        <p className="muted">Belum ada pengajuan. <Link href="/mahasiswa/pengajuan">Ajukan surat →</Link></p>
      ) : (
        items.map((t) => <TicketCard key={t.id} t={t} />)
      )}
    </>
  );
}