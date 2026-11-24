"use client";
import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { api } from "@/lib/api";
import LetterForm from "@/components/LetterForm";

export default function EditLayanan() {
  const { id } = useParams();
  const router = useRouter();
  const [data, setData] = useState(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    api(`/api/staff/letter-types/${id}`).then(setData).catch((e) => setError(e.message));
  }, [id]);

  async function save(payload) {
    setSaving(true);
    setError("");
    try {
      await api(`/api/staff/letter-types/${id}`, { method: "PUT", body: payload });
      router.push("/staf/layanan");
    } catch (e) {
      setError(e.message);
      setSaving(false);
    }
  }

  return (
    <>
      <div className="page-head">
        <Link href="/staf/layanan" className="muted">← Kelola Layanan</Link>
        <h1 style={{ marginTop: 6 }}>Pengaturan Layanan Surat</h1>
      </div>
      {!data ? (
        error ? <div className="alert alert-bad">{error}</div> : <p className="muted">Memuat…</p>
      ) : (
        <LetterForm initial={data} isEdit saving={saving} error={error} onSubmit={save} onCancel={() => router.push("/staf/layanan")} />
      )}
    </>
  );
}