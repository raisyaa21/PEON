"use client";
import { useCallback, useEffect, useRef, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { api, fmt, fetchBlob, openPath } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";

const DECISIONS = [
  ["approving", "Disetujui"],
  ["revision", "Revisi"],
  ["rejected", "Ditolak"],
];

export default function Pemeriksaan() {
  const { id } = useParams();
  const router = useRouter();
  const [t, setT] = useState(null);
  const [decision, setDecision] = useState("approving");
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [preview, setPreview] = useState(null);
  const lastUrl = useRef(null);

  const load = useCallback(async () => {
    try {
      let d = await api(`/api/tickets/${id}`);
      if (d.status === "submitted") {
        await api(`/api/staff/tickets/${id}/start`, { method: "POST" });
        d = await api(`/api/tickets/${id}`);
      }
      setT(d);
    } catch (e) {
      setError(e.message);
    }
  }, [id]);

  useEffect(() => {
    load();
  }, [load]);

  async function show(d) {
    try {
      const blob = await fetchBlob(`/api/documents/${d.id}/file`);
      if (lastUrl.current) URL.revokeObjectURL(lastUrl.current);
      lastUrl.current = URL.createObjectURL(blob);
      setPreview({ url: lastUrl.current, name: `${d.requirement_name}: ${d.file_name}` });
    } catch (e) {
      setError(e.message);
    }
  }

  useEffect(() => {
    if (t && t.documents.length && !preview) show(t.documents[0]);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [t]);

  useEffect(() => () => lastUrl.current && URL.revokeObjectURL(lastUrl.current), []);

  async function save() {
    setError("");
    setSaving(true);
    try {
      await api(`/api/staff/tickets/${id}/status`, { method: "PATCH", body: { status: decision, note } });
      if (decision === "approving") {
        setNote("");
        await load();
        setSaving(false);
      } else {
        router.push("/staf/antrean");
      }
    } catch (e) {
      setError(e.message);
      setSaving(false);
    }
  }

  async function issue() {
    if (!confirm("Terbitkan surat dengan tanda tangan elektronik? Tindakan ini tidak dapat dibatalkan.")) return;
    setError("");
    setSaving(true);
    try {
      await api(`/api/staff/tickets/${id}/issue`, { method: "POST" });
      await load();
    } catch (e) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  }

  if (!t) return error ? <div className="alert alert-bad">{error}</div> : <p className="muted">Memuat…</p>;
  const needNote = decision !== "approving";
  const canDecide = t.status === "reviewing" || t.status === "submitted";

  return (
    <>
      <div className="page-head">
        <Link href="/staf/antrean" className="muted">← Antrean Verifikasi</Link>
        <h1 style={{ marginTop: 6 }}>Pemeriksaan Berkas</h1>
        <span className="muted">Tiket #{t.ticket_no}</span>
      </div>
      {error && <div className="alert alert-bad">{error}</div>}

      <div className="twocol">
        <div>
          <div className="card">
            <div className="row between">
              <b>{t.student_name}</b>
              <StatusBadge status={t.status} />
            </div>
            <dl className="dl">
              <dt>NIM</dt><dd>{t.student_nim}</dd>
              <dt>Layanan Surat</dt><dd>{t.letter_type}</dd>
              <dt>Keperluan</dt><dd>{t.purpose}</dd>
              <dt>Waktu Pengajuan</dt><dd>{fmt(t.created_at)}</dd>
              {t.advisor_status === "recommended" && (
                <><dt>Rekomendasi Dosen PA</dt><dd>Direkomendasikan{t.advisor_note ? `: ${t.advisor_note}` : ""}</dd></>
              )}
              {t.letter_no && <><dt>Nomor Surat</dt><dd>{t.letter_no}</dd></>}
            </dl>
          </div>

          <div className="card">
            <h2>Dokumen Prasyarat</h2>
            {t.documents.map((d) => (
              <div key={d.id} className="slot">
                <span>{d.requirement_name}: {d.file_name} <span className="muted">(v{d.version})</span></span>
                <button className="link" onClick={() => show(d)}>Pratinjau</button>
              </div>
            ))}
          </div>

          {canDecide && (
            <div className="card">
              <h2>Penetapan Hasil Verifikasi</h2>
              <div className="seg">
                {DECISIONS.map(([k, label]) => (
                  <button key={k} type="button" className={decision === k ? "on" : ""} onClick={() => setDecision(k)}>
                    {label}
                  </button>
                ))}
              </div>
              <div className="field">
                <label>Catatan {needNote ? "(wajib)" : "(opsional)"}</label>
                <textarea rows={5} value={note} onChange={(e) => setNote(e.target.value)}
                  placeholder="Contoh: Scan KHS buram, mohon unggah ulang dalam format PDF yang jelas." />
              </div>
              <button className="btn btn-primary btn-block" style={{ padding: 12 }} onClick={save} disabled={saving}>
                {saving ? "Menyimpan…" : "Simpan & Kirim Hasil Verifikasi ▷"}
              </button>
              {decision === "approving" && (
                <p className="hint">Menyetujui akan membuat nomor surat otomatis dan draf surat berkop.</p>
              )}
            </div>
          )}

          {t.status === "approving" && (
            <div className="card">
              <h2>Pengesahan Surat</h2>
              <p>Nomor surat: <b>{t.letter_no}</b></p>
              <div className="row">
                <button className="btn" onClick={() => openPath(`/api/tickets/${id}/letter?kind=draft`).catch((e) => setError(e.message))}>
                  Lihat Draf Surat
                </button>
                <button className="btn btn-primary" onClick={issue} disabled={saving}>
                  {saving ? "Memproses…" : "Terbitkan Surat (TTE)"}
                </button>
              </div>
            </div>
          )}

          {t.status === "completed" && (
            <div className="card">
              <h2>Surat Telah Terbit</h2>
              <p>Nomor surat: <b>{t.letter_no}</b></p>
              <button className="btn btn-primary" onClick={() => openPath(`/api/tickets/${id}/letter?kind=final`).catch((e) => setError(e.message))}>
                Lihat Surat Final
              </button>
            </div>
          )}

          {(t.status === "revision" || t.status === "rejected") && (
            <div className="note-box"><b>Catatan:</b> {t.note}</div>
          )}
        </div>

        <div className="sticky">
          <div className="card">
            <h2>Pratinjau Berkas</h2>
            <div className="muted" style={{ marginBottom: 8 }}>{preview?.name || ""}</div>
            <div className="preview-big">
              {preview ? <iframe src={preview.url} title="Pratinjau" /> : <span>Pilih dokumen untuk dipratinjau</span>}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}