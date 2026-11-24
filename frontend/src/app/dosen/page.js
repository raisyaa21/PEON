"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { api, getUser, fmtDay, advisorState } from "@/lib/api";

export default function DosenHome() {
  const [items, setItems] = useState(null);
  const [user, setUser] = useState(null);
  const [error, setError] = useState("");

  useEffect(() => setUser(getUser()), []);
  useEffect(() => {
    const load = () => api("/api/dosen/requests").then(setItems).catch((e) => setError(e.message));
    load();
    const t = setInterval(load, 10000);
    return () => clearInterval(t);
  }, []);

  return (
    <>
      <div className="page-head">
        {user && <div className="muted">Halo, {user.name} - {user.identifier}</div>}
        <h1>Permohonan Rekomendasi</h1>
      </div>
      {error && <div className="alert alert-bad">{error}</div>}
      <div className="card table-wrap">
        {!items ? (
          <p className="muted">Memuat…</p>
        ) : items.length === 0 ? (
          <p className="muted">Belum ada permohonan dari mahasiswa bimbingan Anda.</p>
        ) : (
          <table>
            <thead>
              <tr><th>Mahasiswa</th><th>NIM</th><th>Jenis Surat</th><th>Tanggal</th><th>Status</th><th></th></tr>
            </thead>
            <tbody>
              {items.map((t) => {
                const [label, cls] = advisorState(t);
                return (
                  <tr key={t.id}>
                    <td><b>{t.student_name}</b></td>
                    <td>{t.student_nim}</td>
                    <td>{t.letter_type}</td>
                    <td>{fmtDay(t.created_at)}</td>
                    <td><span className={`chip ${cls}`}>{label}</span></td>
                    <td><Link href={`/dosen/permohonan/${t.id}`} className="btn btn-primary btn-sm">Lihat Detail</Link></td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>
    </>
  );
}