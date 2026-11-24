"use client";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api, fmtDay } from "@/lib/api";

export default function Verifikasi() {
  const { code } = useParams();
  const [d, setD] = useState(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api(`/api/verify/${code}`).then(setD).catch((e) => setError(e.message));
  }, [code]);

  return (
    <div className="center">
      <div className="login-card">
        {!d && !error && <p className="muted" style={{ textAlign: "center" }}>Memeriksa dokumen…</p>}
        {error && (
          <>
            <div className="login-logo" style={{ background: "var(--bad)" }}>✕</div>
            <h2 style={{ textAlign: "center" }}>Dokumen Tidak Valid</h2>
            <p className="muted" style={{ textAlign: "center" }}>{error}</p>
          </>
        )}
        {d && (
          <>
            <div className="login-logo" style={{ background: "var(--ok)" }}>✓</div>
            <h2 style={{ textAlign: "center" }}>Dokumen Sah</h2>
            <p className="muted" style={{ textAlign: "center" }}>Surat ini tercatat dan diterbitkan melalui Portal Kampus.</p>
            <dl className="dl">
              <dt>Nomor Surat</dt><dd>{d.letter_no}</dd>
              <dt>Jenis Surat</dt><dd>{d.letter_type}</dd>
              <dt>Atas Nama</dt><dd>{d.student_name} ({d.student_nim})</dd>
              <dt>Ditandatangani Oleh</dt><dd>{d.signed_by}</dd>
              <dt>Tanggal Terbit</dt><dd>{fmtDay(d.issued_at)}</dd>
              <dt>Hash Dokumen</dt><dd>{d.hash}…</dd>
            </dl>
          </>
        )}
      </div>
    </div>
  );
}