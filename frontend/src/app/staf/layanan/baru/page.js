"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api } from "@/lib/api";
import LetterForm, { EMPTY_LETTER } from "@/components/LetterForm";

export default function BaruLayanan() {
  const router = useRouter();
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function save(payload) {
    setSaving(true);
    setError("");
    try {
      await api("/api/staff/letter-types", { method: "POST", body: payload });
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
        <h1 style={{ marginTop: 6 }}>Tambah Layanan Surat</h1>
      </div>
      <LetterForm initial={EMPTY_LETTER} saving={saving} error={error} onSubmit={save} onCancel={() => router.push("/staf/layanan")} />
    </>
  );
}