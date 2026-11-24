"use client";
import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { api, fmtDay, openDocument, advisorState } from "@/lib/api";

export default function DetailPermohonan() {
  const { id } = useParams();
  const router = useRouter();
  const [d, setD] = useState(null);
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api(`/api/dosen/requests/${id}`).then(setD).catch((e) => setError(e.message));
  }, [id]);

  async function decide(decision) {
    setError("");
    setSaving(true);
    try {
      await api(`/api/dosen/requests/${id}/decision`, { method: "POST", body: { decision, note } });
      router.push("/dosen");
    } catch (e) {
      setError(e.message);
      setSaving(false);
    }
  }

  if (!d) return error ? <div className="alert alert-bad">{error}</div> : <p className="muted">Memuat…</p>;
  const pending = d.status === "awaiting_recommendation";
  const [label, cls] = advisorState(d);

  return (
    <>
      <div className="page-head">
        <Link href="/dosen" className="muted">← Kembali</Link>
        <h1 style={{ marginTop: 6 }}>Detail Permohonan</h1>
        <span className="muted">#{d.ticket_no}</span>
      </div>
      {error && <div className="alert alert-bad">{error}</div>}

      <div className="twocol">
        <div>
          <div className="card">
            <h2>Rincian Pengajuan</h2>
            <dl className="dl">
              <dt>Mahasiswa</dt><dd>{d.student_name} ({d.student_nim})</dd>
              <dt>Program Studi</dt><dd>{d.program} • Semester {d.semester}</dd>
              <dt>Jenis Surat</dt><dd>{d.letter_type}</dd>
              <dt>Tanggal Pengajuan</dt><dd>{fmtDay(d.created_at)}</dd>
              <dt>Keperluan</dt><dd>{d.purpose}</dd>
              <dt>Dokumen Pendukung</dt>
              <dd>
                {d.documents.map((x) => (
                  <div key={x.id} className="row" style={{ marginBottom: 4 }}>
                    <span>{x.requirement_name}: {x.file_name}</span>
                    <button className="link" onClick={() => openDocument(x.id).catch((e) => setError(e.message))}>Lihat</button>
                  </div>
                ))}
              </dd>
            </dl>
          </div>

          <div className="card">
            <h2>Pratinjau Riwayat Akademik</h2>
            <table>
              <thead><tr><th>Semester</th><th>IPS</th><th>SKS</th></tr></thead>
              <tbody>
                {d.records.map((r) => (
                  <tr key={r.semester}><td>{r.semester}</td><td>{r.ips.toFixed(2)}</td><td>{r.sks}</td></tr>
                ))}
                <tr><td><b>IPK</b></td><td colSpan={2}><b>{d.ipk ? d.ipk.toFixed(2) : "-"}</b></td></tr>
              </tbody>
            </table>
          </div>
        </div>

        <div className="card">
          <h2>Catatan Masukan Akademik</h2>
          {pending ? (
            <>
              <textarea rows={10} value={note} onChange={(e) => setNote(e.target.value)} placeholder="Tulis catatan masukan akademik…" />
              <p className="hint">Catatan wajib diisi jika permohonan ditolak.</p>
              <div className="row" style={{ justifyContent: "flex-end" }}>
                <button className="btn btn-danger" disabled={saving} onClick={() => decide("rejected")}>Tolak Permohonan</button>
                <button className="btn btn-primary" disabled={saving} onClick={() => decide("recommended")}>Rekomendasikan</button>
              </div>
            </>
          ) : (
            <>
              <span className={`chip ${cls}`}>{label}</span>
              <p>{d.advisor_note || <span className="muted">Tanpa catatan.</span>}</p>
            </>
          )}
        </div>
      </div>
    </>
  );
}