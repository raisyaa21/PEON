"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { api, getUser } from "@/lib/api";
import ArchiveItem from "@/components/ArchiveItem";

export default function MahasiswaHome() {
  const [q, setQ] = useState("");
  const [items, setItems] = useState([]);
  const [letters, setLetters] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [user, setUser] = useState(null);

  useEffect(() => setUser(getUser()), []);

  useEffect(() => {
    api("/api/archive").then(setLetters).catch(() => {});
  }, []);

  useEffect(() => {
    const t = setTimeout(() => {
      api(`/api/letter-types?q=${encodeURIComponent(q)}`)
        .then(setItems)
        .catch((e) => setError(e.message))
        .finally(() => setLoading(false));
    }, 300);
    return () => clearTimeout(t);
  }, [q]);

  return (
    <>
      <div className="page-head">
        <h1>Beranda Mahasiswa</h1>
        {user && <div className="muted">{user.name} • {user.identifier}</div>}
      </div>

      <div className="field">
        <input
          className="input"
          style={{ background: "#fff", border: "1px solid var(--border)" }}
          placeholder="Cari jenis surat…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>

      <h2>Menu Layanan Surat</h2>
      {error && <div className="alert alert-bad">{error}</div>}
      {loading ? (
        <p className="muted">Memuat layanan…</p>
      ) : items.length === 0 ? (
        <p className="muted">Tidak ada jenis surat yang cocok dengan “{q}”.</p>
      ) : (
        <div className="menu-grid">
          {items.map((l) => (
            <div className="svc" key={l.id}>
              <b>{l.name}</b>
              <span className="muted">Syarat: {l.requirements.map((r) => r.name).join(", ")}</span>
              <Link href={`/mahasiswa/pengajuan?jenis=${l.id}`} className="btn btn-primary">
                Ajukan
              </Link>
            </div>
          ))}
        </div>
      )}

      {letters.length > 0 && (
        <>
          <h2 style={{ marginTop: 28 }}>Surat Siap Diunduh</h2>
          {letters.slice(0, 3).map((it) => (
            <ArchiveItem key={it.ticket_id} item={it} />
          ))}
          {letters.length > 3 && <Link href="/mahasiswa/riwayat">Lihat semua di Riwayat →</Link>}
        </>
      )}
    </>
  );
}