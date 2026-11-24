"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";

const SET_FIELDS = [
  ["kop_1", "Kop baris 1"],
  ["kop_2", "Kop baris 2 (instansi)"],
  ["kop_3", "Kop baris 3 (fakultas)"],
  ["address", "Alamat"],
  ["city", "Kota pada tanggal surat"],
  ["signer_name", "Nama penandatangan"],
  ["signer_title", "Jabatan penandatangan"],
];

function KopSettings() {
  const [s, setS] = useState(null);
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");

  useEffect(() => {
    api("/api/staff/settings").then(setS).catch((e) => setErr(e.message));
  }, []);

  async function save() {
    setMsg("");
    setErr("");
    try {
      await api("/api/staff/settings", { method: "PUT", body: s });
      setMsg("Pengaturan kop dan penandatangan tersimpan.");
    } catch (e) {
      setErr(e.message);
    }
  }

  if (!s) return null;
  return (
    <div className="card" style={{ marginTop: 24 }}>
      <h2>Kop Surat & Penandatangan</h2>
      {err && <div className="alert alert-bad">{err}</div>}
      {msg && <div className="alert alert-ok">{msg}</div>}
      {SET_FIELDS.map(([k, label]) => (
        <div className="field" key={k}>
          <label>{label}</label>
          <input className="input" value={s[k] || ""} onChange={(e) => setS({ ...s, [k]: e.target.value })} />
        </div>
      ))}
      <button className="btn btn-primary" onClick={save}>Simpan Pengaturan</button>
    </div>
  );
}

export default function Layanan() {
  const [items, setItems] = useState(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api("/api/staff/letter-types").then(setItems).catch((e) => setError(e.message));
  }, []);

  return (
    <>
      <div className="page-head row between">
        <div>
          <div className="eyebrow">Katalog Administratif</div>
          <h1>Kelola Layanan & Template</h1>
        </div>
        <Link href="/staf/layanan/baru" className="btn btn-primary">+ Tambah Layanan</Link>
      </div>
      {error && <div className="alert alert-bad">{error}</div>}

      {!items ? (
        <p className="muted">Memuat…</p>
      ) : (
        <div className="menu-grid">
          {items.map((l) => (
            <div className="svc" key={l.id}>
              <div className="row between">
                <span className="chip code">{l.code}</span>
                <span className={`chip ${l.active ? "ok" : "wait"}`}>{l.active ? "Aktif" : "Nonaktif"}</span>
              </div>
              <b>{l.name}</b>
              <span className="muted">Format nomor: {l.number_format}</span>
              {l.requires_recommendation && <span className="muted">✓ Perlu rekomendasi Dosen PA</span>}
              <span className="muted">Syarat: {l.requirements.map((r) => r.name).join(", ")}</span>
              <Link href={`/staf/layanan/${l.id}`} className="btn btn-primary">Edit Layanan & Template</Link>
            </div>
          ))}
        </div>
      )}
      <KopSettings />
    </>
  );
}