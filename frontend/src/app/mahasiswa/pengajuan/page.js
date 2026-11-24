"use client";
import { Suspense, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { api } from "@/lib/api";
import FileField from "@/components/FileField";

function Form() {
  const router = useRouter();
  const params = useSearchParams();
  const [types, setTypes] = useState([]);
  const [typeId, setTypeId] = useState(params.get("jenis") || "");
  const [purpose, setPurpose] = useState("");
  const [files, setFiles] = useState({});
  const [errors, setErrors] = useState({});
  const [preview, setPreview] = useState(null);
  const [confirmed, setConfirmed] = useState(false);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [done, setDone] = useState(null);

  useEffect(() => {
    api("/api/letter-types")
      .then((list) => {
        setTypes(list);
        setTypeId((cur) => cur || String(list[0]?.id || ""));
      })
      .catch((e) => setError(e.message));
  }, []);

  const lt = types.find((t) => String(t.id) === String(typeId));

  const previewUrl = useMemo(() => (preview ? URL.createObjectURL(preview) : null), [preview]);
  useEffect(() => () => previewUrl && URL.revokeObjectURL(previewUrl), [previewUrl]);

  function changeType(v) {
    setTypeId(v);
    setFiles({});
    setErrors({});
    setPreview(null);
  }

  function pick(reqId, file, msg) {
    setFiles((p) => ({ ...p, [reqId]: file || undefined }));
    setErrors((p) => ({ ...p, [reqId]: msg }));
    if (file) setPreview(file);
  }

  async function submit() {
    setError("");
    if (!purpose.trim()) return setError("Keperluan pembuatan surat wajib diisi.");
    const missing = lt.requirements.filter((r) => !files[r.id]);
    if (missing.length) {
      setErrors((p) => {
        const n = { ...p };
        missing.forEach((r) => (n[r.id] = n[r.id] || "Dokumen ini wajib diunggah."));
        return n;
      });
      return setError("Lengkapi seluruh dokumen prasyarat sebelum mengirim.");
    }
    setSubmitting(true);
    try {
      const form = new FormData();
      form.append("letter_type_id", lt.id);
      form.append("purpose", purpose);
      form.append("confirmed", "true");
      lt.requirements.forEach((r) => form.append(`req_${r.id}`, files[r.id]));
      const res = await api("/api/tickets", { method: "POST", form });
      setDone(res);
    } catch (e) {
      setError(e.message);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <>
      <div className="page-head">
        <div className="eyebrow">Layanan Akademik</div>
        <h1>Pengajuan Surat</h1>
      </div>

      {error && <div className="alert alert-bad">{error}</div>}

      <div className="card">
        <div className="field">
          <label>Jenis Surat</label>
          <select value={typeId} onChange={(e) => changeType(e.target.value)}>
            {types.map((t) => (
              <option key={t.id} value={t.id}>{t.name}</option>
            ))}
          </select>
        </div>
        <div className="field">
          <label>Keperluan Pembuatan Surat</label>
          <textarea
            rows={3}
            value={purpose}
            onChange={(e) => setPurpose(e.target.value)}
            placeholder="Contoh: Magang di Divisi Teknologi Informasi, Periode September–November 2026."
          />
        </div>

        {lt && (
          <>
            <label className="lbl">
              Dokumen prasyarat <span className="muted">(PDF atau JPG, maks. 5 MB per berkas)</span>
            </label>
            {lt.requirements.map((r) => (
              <FileField
                key={r.id}
                label={`${r.name} · Seret berkas atau pilih file`}
                file={files[r.id]}
                error={errors[r.id]}
                onPick={(f, msg) => pick(r.id, f, msg)}
                onView={() => setPreview(files[r.id])}
              />
            ))}
          </>
        )}
      </div>

      <div className="card">
        <h2>Pratinjau dan konfirmasi</h2>
        <div className="preview">
          {!preview ? (
            <span>Pilih berkas untuk melihat pratinjau</span>
          ) : preview.type === "image/jpeg" ? (
            <img src={previewUrl} alt={preview.name} />
          ) : (
            <iframe src={previewUrl} title={preview.name} />
          )}
        </div>
        <label className="check-row">
          <input type="checkbox" checked={confirmed} onChange={(e) => setConfirmed(e.target.checked)} />
          Saya menyatakan data dan berkas yang diunggah sudah benar.
        </label>
        <button
          className="btn btn-primary btn-block"
          style={{ padding: 12 }}
          disabled={!confirmed || submitting || !lt}
          onClick={submit}
        >
          {submitting ? "Mengirim…" : "▷ Kirim Pengajuan"}
        </button>
      </div>

      {done && (
        <div className="overlay">
          <div className="modal">
            <div className="ok-ic">✓</div>
            <div className="eyebrow">Layanan Akademik</div>
            <h2>Pengajuan Berhasil Dikirim!</h2>
            <div className="tk">Nomor Tiket: #{done.ticket_no}</div>
            <p className="muted">
              Permohonan {lt?.name} Anda telah tersimpan ke dalam sistem. Status permohonan dapat dipantau melalui menu Status.
            </p>
            <button className="btn btn-primary btn-block" onClick={() => router.push("/mahasiswa/status")}>
              Lacak Status Pengajuan
            </button>
            <p><Link href="/mahasiswa" className="muted">Kembali ke Beranda</Link></p>
          </div>
        </div>
      )}
    </>
  );
}

export default function PengajuanPage() {
  return (
    <Suspense fallback={<p className="muted">Memuat…</p>}>
      <Form />
    </Suspense>
  );
}