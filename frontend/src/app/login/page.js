"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, saveSession, getUser, homeFor } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [type, setType] = useState("mahasiswa");
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const u = getUser();
    if (u) router.replace(homeFor(u.role));
  }, [router]);

  async function onSubmit(e) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await api("/api/auth/login", {
        method: "POST",
        body: { identifier, password, type },
      });
      saveSession(res.token, res.user);
      router.replace(homeFor(res.user.role));
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  const isStudent = type === "mahasiswa";

  return (
    <div className="center">
      <form className="login-card" onSubmit={onSubmit}>
        <div className="login-logo">▣</div>
        <div style={{ textAlign: "center" }}>
          <b>Portal Akademik</b>
          <div className="muted">Masuk menggunakan NIM/NIP</div>
        </div>

        <div className="tabs">
          <button type="button" className={`tab ${isStudent ? "on" : ""}`} onClick={() => setType("mahasiswa")}>
            Mahasiswa (NIM)
          </button>
          <button type="button" className={`tab ${!isStudent ? "on" : ""}`} onClick={() => setType("pegawai")}>
            Dosen / Tendik (NIP)
          </button>
        </div>

        {error && (
          <div className="alert alert-bad">
            <b>⚠ Gagal Masuk</b>
            <div>{error}</div>
          </div>
        )}

        <div className="field">
          <label htmlFor="id">{isStudent ? "NIM/NIP" : "Nomor Identitas (NIP)"}</label>
          <input
            id="id"
            className={`input ${error ? "bad" : ""}`}
            placeholder="Contoh: 2108561001 atau 19850315…"
            value={identifier}
            onChange={(e) => setIdentifier(e.target.value)}
            required
          />
          <div className="hint">ⓘ Masukkan 10-18 digit angka resmi kampus</div>
        </div>

        <div className="field" style={{ marginBottom: 6 }}>
          <label htmlFor="pw">Kata Sandi</label>
          <input
            id="pw"
            type="password"
            className={`input ${error ? "bad" : ""}`}
            placeholder="Masukkan kata sandi portal Anda"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </div>

        <div className="login-opts">
          <label className="row" style={{ gap: 6 }}>
            <input type="checkbox" /> Ingat Saya
          </label>
          <a href="#">Lupa Kata Sandi?</a>
        </div>

        <button className="btn btn-grad btn-block" style={{ padding: 12 }} disabled={loading}>
          {loading ? "Memproses…" : "Masuk ke Portal ⇥"}
        </button>

        <div className="helpbox">
          <b style={{ color: "var(--text)" }}>Mengalami kendala akun?</b>
          <div>Hubungi Helpdesk IT Kampus terpadu untuk reset kredensial atau panduan KRS.</div>
        </div>

        <div className="demo">
          <b>Akun demo</b> (sandi: password123)<br />
          Mahasiswa: 255150400111001 · Staf: 198501012010 · Dosen: 197803152005
        </div>
      </form>
    </div>
  );
}