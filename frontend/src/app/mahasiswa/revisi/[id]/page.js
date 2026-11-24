"use client";
import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { api, openDocument } from "@/lib/api";
import FileField from "@/components/FileField";

export default function RevisiPage() {
  const { id } = useParams();
  const router = useRouter();
  const [t, setT] = useState(null);
  const [files, setFiles] = useState({});
  const [errors, setErrors] = useState({});
  const [error, setError] = useState("");
  const [sending, setSending] = useState(false);

  useEffect(() => {
    api(`/api/tickets/${id}`).then(setT).catch((e) => setError(e.message));
  }, [id]);

  async function send() {
    setError("");
    const form = new FormData();
    let count = 0;
    t.requirements.forEach((r) => {
      if (files[r.id]) {
        form.append(`req_${r.id}`, files[r.id]);
        count++;
      }
    });
    if (!count) return setError("Pilih minimal satu dokumen pengganti.");
    setSending(true);
    try {
      await api(`/api/tickets/${id}/resubmit`, { method: "POST", form });
      router.push("/mahasiswa/status");
    } catch (e) {
      setError(e.message);
      setSending(false);
    }
  }

  if (!t) return error ? <div className="alert alert-bad">{error}</div> : <p className="muted">Memuat…</p>;

  return (
    <>
      <div className="page-head">
        <Link href="/mahasiswa/status" className="muted">← Kembali ke Status</Link>
        <div className="eyebrow" style={{ marginTop: 6 }}>Layanan Akademik</div>
        <h1>Revisi Dokumen</h1>
      </div>

      {error && <div className="alert alert-bad">{error}</div>}

      {t.status !== "revision" ? (
        <div className="alert alert-warn">Pengajuan ini sedang tidak meminta revisi.</div>
      ) : (
        <>
          <div className="card">
            <div className="row between">
              <div><div className="muted">Nomor Surat</div><b>#{t.ticket_no}</b></div>
              <div><div className="muted">Jenis Permohonan</div><b>{t.letter_type}</b></div>
            </div>
            <h2 style={{ marginTop: 16 }}>Catatan Perbaikan</h2>
            <div className="note-box">{t.note}</div>
          </div>

          <div className="card">
            <div className="eyebrow">Berkas Terkait</div>
            <h2>Unggah Berkas Baru</h2>
            {t.requirements.map((r) => {
              const cur = t.documents.find((d) => d.requirement_id === r.id);
              return (
                <div key={r.id}>
                  <FileField
                    label={`${r.name} · Seret berkas atau pilih file · PDF/JPG, maks. 5 MB`}
                    file={files[r.id]}
                    error={errors[r.id]}
                    onPick={(f, msg) => {
                      setFiles((p) => ({ ...p, [r.id]: f || undefined }));
                      setErrors((p) => ({ ...p, [r.id]: msg }));
                    }}
                  />
                  {cur && (
                    <div className="muted" style={{ margin: "-4px 0 10px" }}>
                      Berkas saat ini: {cur.file_name} (v{cur.version}){" "}
                      <button className="link" onClick={() => openDocument(cur.id).catch((e) => setError(e.message))}>
                        Lihat
                      </button>
                    </div>
                  )}
                </div>
              );
            })}
            <p className="muted">Data formulir dan berkas lain tidak perlu diisi ulang.</p>
            <button className="btn btn-primary btn-block" style={{ padding: 12 }} onClick={send} disabled={sending}>
              {sending ? "Mengirim…" : "▷ Kirim Dokumen Revisi"}
            </button>
          </div>
        </>
      )}
    </>
  );
}