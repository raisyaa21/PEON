"use client";
import { useCallback, useEffect, useState } from "react";
import { api, getUser } from "@/lib/api";
import Icon from "@/components/Icon";

const ROLE_OPTS = [
  ["dosen_pa", "Dosen Pembimbing Akademik (Dosen PA)"],
  ["dosen_skripsi", "Dosen Pembimbing Skripsi / Tugas Akhir (Dospem)"],
  ["dosen_dpl", "Dosen Pembimbing Lapangan (DPL)"],
  ["staf", "Staf Administrasi & Operasional"],
];

const pad = (n) => String(n).padStart(2, "0");
function logTime(s) {
  const d = new Date(s);
  const tgl = d.toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" });
  return `${tgl}, ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function AuditTab() {
  const [rows, setRows] = useState(null);
  const [error, setError] = useState("");

  useEffect(() => {
    const load = () => api("/api/staff/audit").then(setRows).catch((e) => setError(e.message));
    load();
    const t = setInterval(load, 15000);
    return () => clearInterval(t);
  }, []);

  return (
    <div className="panel">
      {error && <div className="alert alert-bad">{error}</div>}
      {!rows ? (
        <p className="muted">Memuat…</p>
      ) : rows.length === 0 ? (
        <p className="muted">Belum ada aktivitas tercatat.</p>
      ) : (
        rows.map((l) => (
          <div className="logline" key={l.id}>
            <span>
              {logTime(l.created_at)} - <b>{l.actor_name || "Sistem Otomatis"}</b> {l.action}
            </span>
            <b className="id">LOG-{String(l.id).padStart(5, "0")}</b>
          </div>
        ))
      )}
    </div>
  );
}

function AccessTab() {
  const me = getUser();
  const [users, setUsers] = useState(null);
  const [q, setQ] = useState("");
  const [edit, setEdit] = useState(null);
  const [pick, setPick] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const load = useCallback(() => {
    api(`/api/staff/users?q=${encodeURIComponent(q)}`).then(setUsers).catch((e) => setError(e.message));
  }, [q]);

  useEffect(() => {
    const t = setTimeout(load, 300);
    return () => clearTimeout(t);
  }, [load]);

  const options = (u) =>
    u.role === "mahasiswa"
      ? [["active", "Akun Aktif (Mahasiswa)"], ["inactive", "Nonaktifkan Akun (Nonaktif)"]]
      : [...ROLE_OPTS, ["inactive", "Nonaktifkan Akun (Nonaktif)"]];

  function open(u) {
    setEdit(u);
    setError("");
    if (!u.active) return setPick("inactive");
    if (u.role === "mahasiswa") return setPick("active");
    setPick((ROLE_OPTS.find(([, label]) => label === u.position) || ["staf"])[0]);
  }

  async function save() {
    setSaving(true);
    setError("");
    try {
      await api(`/api/staff/users/${edit.id}/access`, { method: "PATCH", body: { access: pick } });
      setEdit(null);
      load();
    } catch (e) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="panel">
      <div className="access-head">
        <div>
          <h2 style={{ margin: 0 }}>
            Daftar Pengguna & Hak Akses{" "}
            {users && <span className="chip code">{users.length} Akun Terdaftar</span>}
          </h2>
          <p className="muted" style={{ margin: "4px 0 0" }}>
            Kelola peran akun dan otorisasi modul operasional akademik sivitas kampus.
          </p>
        </div>
        <div className="search-box">
          <Icon name="search" size={15} />
          <input className="input" placeholder="Cari nama atau NIP…" value={q} onChange={(e) => setQ(e.target.value)} />
        </div>
      </div>

      {error && !edit && <div className="alert alert-bad">{error}</div>}
      {!users ? (
        <p className="muted">Memuat…</p>
      ) : (
        users.map((u) => (
          <div className="urow" key={u.id}>
            <div className="row" style={{ flexWrap: "nowrap" }}>
              <span className="ub"><Icon name="user" size={16} /></span>
              <div>
                <b>{u.name}</b>{" "}
                <span className={`chip ${u.active ? "ok" : "no"}`}>• {u.active ? "Aktif" : "Nonaktif"}</span>
                <div className="muted">
                  {u.position || (u.role === "mahasiswa" ? "Mahasiswa" : u.role)} · {u.role === "mahasiswa" ? "NIM" : "NIP"}: {u.identifier}
                </div>
              </div>
            </div>
            <button className="btn btn-sm btn-ico" disabled={me?.id === u.id} onClick={() => open(u)}>
              <Icon name="edit" size={13} /> Ubah
            </button>
          </div>
        ))
      )}

      <div className="panel-foot">
        <span>ⓘ Setiap perubahan hak akses langsung dicatat ke dalam audit trail institusi.</span>
        <span>{users ? `${users.length} akun` : ""}</span>
      </div>

      {edit && (
        <div className="overlay" onClick={() => !saving && setEdit(null)}>
          <div className="modal left" onClick={(e) => e.stopPropagation()}>
            <div className="eyebrow">Otorisasi & Peran</div>
            <h2>Ubah Hak Akses & Peran</h2>
            <p className="muted">
              Kelola hak peran untuk <b>{edit.name}</b> ({edit.identifier})
            </p>
            {error && <div className="alert alert-bad">{error}</div>}
            {options(edit).map(([k, label]) => (
              <label key={k} className={`opt ${pick === k ? "on" : ""} ${k === "inactive" ? "danger" : ""}`}>
                <span>{label}</span>
                <input type="radio" name="access" checked={pick === k} onChange={() => setPick(k)} />
              </label>
            ))}
            <div className="row" style={{ justifyContent: "flex-end", marginTop: 12 }}>
              <button className="btn" onClick={() => setEdit(null)} disabled={saving}>Batal</button>
              <button className="btn btn-primary" onClick={save} disabled={saving}>
                {saving ? "Menyimpan…" : "✓ Simpan Perubahan"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default function Manajemen() {
  const [tab, setTab] = useState("log");
  return (
    <>
      <div className="page-head"><h1>Manajemen Hak Akses & Audit Log</h1></div>
      <div className="seg-tabs">
        <button className={tab === "log" ? "on" : ""} onClick={() => setTab("log")}>
          <Icon name="clock" size={15} /> Audit Log
        </button>
        <button className={tab === "akses" ? "on" : ""} onClick={() => setTab("akses")}>
          <Icon name="users" size={15} /> Hak Akses
        </button>
      </div>
      {tab === "log" ? <AuditTab /> : <AccessTab />}
    </>
  );
}