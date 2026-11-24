"use client";
import { useState } from "react";

const ROMAN = ["", "I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"];
const PLACEHOLDERS = ["{{nama}}", "{{nim}}", "{{prodi}}", "{{semester}}", "{{keperluan}}", "{{nomor}}", "{{tanggal}}", "{{jenis}}", "{{jabatan}}"];

export const EMPTY_LETTER = {
  code: "",
  name: "",
  description: "",
  active: true,
  requires_recommendation: false,
  number_format: "{NO}/{KODE}/{JUR}/{BLN}/{THN}",
  template_body:
    "Yang bertanda tangan di bawah ini menerangkan bahwa:\n\nNama : {{nama}}\nNIM : {{nim}}\nProgram Studi : {{prodi}}\n\nmengajukan {{jenis}} untuk keperluan: {{keperluan}}.\n\nDemikian surat ini dibuat untuk dipergunakan sebagaimana mestinya.",
  requirements: [{ id: 0, name: "Scan KTM" }],
};

function sample(f, code) {
  const d = new Date();
  return f
    .replaceAll("{NO}", "001")
    .replaceAll("{KODE}", code || "KODE")
    .replaceAll("{JUR}", "SI")
    .replaceAll("{BLN}", ROMAN[d.getMonth() + 1])
    .replaceAll("{THN}", String(d.getFullYear()));
}

export default function LetterForm({ initial, isEdit, saving, error, onSubmit, onCancel }) {
  const [f, setF] = useState(initial);
  const set = (k, v) => setF((p) => ({ ...p, [k]: v }));
  const setReq = (i, name) =>
    setF((p) => ({ ...p, requirements: p.requirements.map((r, x) => (x === i ? { ...r, name } : r)) }));
  const addReq = () => setF((p) => ({ ...p, requirements: [...p.requirements, { id: 0, name: "" }] }));
  const delReq = (i) => setF((p) => ({ ...p, requirements: p.requirements.filter((_, x) => x !== i) }));

  return (
    <>
      {error && <div className="alert alert-bad">{error}</div>}

      <div className="card">
        <h2>Pengaturan & Template Surat</h2>
        <div className="field">
          <label>Nama Layanan Surat</label>
          <input className="input" value={f.name} onChange={(e) => set("name", e.target.value)} />
        </div>
        <div className="field">
          <label>Deskripsi Singkat</label>
          <input className="input" value={f.description} onChange={(e) => set("description", e.target.value)} />
        </div>
        <div className="twocol">
          <div className="field">
            <label>Kode Prefix</label>
            <input className="input" value={f.code} onChange={(e) => set("code", e.target.value.toUpperCase())} placeholder="KM-PKL" />
          </div>
          <div className="field">
            <label>Format Penomoran</label>
            <input className="input" value={f.number_format} onChange={(e) => set("number_format", e.target.value)} />
            <div className="hint">
              Token: {"{NO}"} {"{KODE}"} {"{JUR}"} {"{BLN}"} {"{THN}"}. Contoh hasil: <b>{sample(f.number_format, f.code)}</b>
            </div>
          </div>
        </div>
        <label className="check-row">
          <input type="checkbox" checked={f.requires_recommendation} onChange={(e) => set("requires_recommendation", e.target.checked)} />
          Perlu rekomendasi Dosen PA sebelum diproses staf
        </label>
        {isEdit && (
          <label className="check-row">
            <input type="checkbox" checked={f.active} onChange={(e) => set("active", e.target.checked)} />
            Layanan aktif (tampil di menu mahasiswa)
          </label>
        )}
      </div>

      <div className="card">
        <h2>Persyaratan Wajib Mahasiswa</h2>
        {f.requirements.map((r, i) => (
          <div className="row" style={{ marginBottom: 8, flexWrap: "nowrap" }} key={i}>
            <input className="input" value={r.name} onChange={(e) => setReq(i, e.target.value)} placeholder="Nama dokumen" />
            <button type="button" className="btn btn-sm btn-danger" onClick={() => delReq(i)}>Hapus</button>
          </div>
        ))}
        <button type="button" className="btn btn-sm" onClick={addReq}>+ Tambah persyaratan</button>
        <p className="hint">Persyaratan yang sudah dipakai pengajuan tidak dapat dihapus.</p>
      </div>

      <div className="card">
        <h2>Template Naskah Surat</h2>
        <p className="hint" style={{ marginTop: -6 }}>
          Placeholder yang tersedia: {PLACEHOLDERS.join("  ")}
        </p>
        <textarea rows={14} value={f.template_body} onChange={(e) => set("template_body", e.target.value)} style={{ fontFamily: "monospace" }} />
      </div>

      <div className="row">
        <button type="button" className="btn" onClick={onCancel} disabled={saving}>Batal</button>
        <button
          type="button"
          className="btn btn-primary"
          disabled={saving}
          onClick={() =>
            onSubmit({ ...f, requirements: f.requirements.map((r) => ({ id: r.id || 0, name: r.name })) })
          }
        >
          {saving ? "Menyimpan…" : "Simpan & Terapkan"}
        </button>
      </div>
    </>
  );
}