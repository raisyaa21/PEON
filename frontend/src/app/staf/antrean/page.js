"use client";
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { api, fmt } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";

export default function Antrean() {
  const [tickets, setTickets] = useState(null);
  const [filter, setFilter] = useState("");
  const [error, setError] = useState("");

  const load = useCallback(
    () => api(`/api/staff/tickets?status=${filter}`).then(setTickets).catch((e) => setError(e.message)),
    [filter]
  );

  useEffect(() => {
    load();
    const t = setInterval(load, 10000);
    return () => clearInterval(t);
  }, [load]);

  return (
    <>
      <div className="page-head">
        <h1>Antrean Verifikasi</h1>
        <p className="muted">Periksa berkas dan tentukan keputusan pengajuan mahasiswa.</p>
      </div>
      {error && <div className="alert alert-bad">{error}</div>}

      <div className="field" style={{ maxWidth: 260 }}>
        <select value={filter} onChange={(e) => setFilter(e.target.value)}>
          <option value="">Semua status</option>
          <option value="submitted">Pengajuan Diterima</option>
          <option value="reviewing">Pemeriksaan Berkas</option>
          <option value="revision">Perlu Revisi</option>
          <option value="approving">Pengesahan Surat</option>
          <option value="rejected">Ditolak</option>
          <option value="completed">Selesai Diterbitkan</option>
        </select>
      </div>

      {!tickets ? (
        <p className="muted">Memuat…</p>
      ) : tickets.length === 0 ? (
        <p className="muted">Tidak ada pengajuan.</p>
      ) : (
        tickets.map((t) => (
          <div className="qcard" key={t.id}>
            <div className="row between">
              <div className="row">
                <span className="avatar">{t.student_name.split(" ").map((w) => w[0]).slice(0, 2).join("")}</span>
                <div>
                  <b>{t.student_name}</b>
                  <div className="muted">NIM: {t.student_nim} · {fmt(t.created_at)}</div>
                </div>
              </div>
              <StatusBadge status={t.status} />
            </div>
            <p style={{ fontWeight: 600, margin: "10px 0" }}>{t.letter_type}</p>
            <Link href={`/staf/antrean/${t.id}`} className="btn btn-primary btn-block">Periksa Berkas →</Link>
          </div>
        ))
      )}
    </>
  );
}