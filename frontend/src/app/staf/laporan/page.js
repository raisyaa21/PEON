"use client";
import { useEffect, useState } from "react";
import { api, downloadPath } from "@/lib/api";
import Icon from "@/components/Icon";

const MONTHS = ["Seluruh Tahun", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"];

export default function Laporan() {
  const now = new Date();
  const [month, setMonth] = useState(now.getMonth() + 1);
  const [year, setYear] = useState(now.getFullYear());
  const [data, setData] = useState(null);
  const [error, setError] = useState("");
  const [exportErr, setExportErr] = useState("");

  async function apply() {
    setError("");
    try {
      setData(await api(`/api/staff/reports?month=${month}&year=${year}`));
    } catch (e) {
      setError(e.message);
    }
  }

  useEffect(() => {
    apply();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function exportPdf() {
    setExportErr("");
    try {
      await downloadPath(
        `/api/staff/reports/export?month=${month}&year=${year}`,
        `laporan-persuratan-${year}-${month || "tahunan"}.pdf`
      );
    } catch {
      setExportErr("Dokumen PDF tidak dapat diunduh saat ini.");
    }
  }

  const years = [0, 1, 2, 3].map((i) => now.getFullYear() - i);
  const total = data?.total || 0;
  const pct = (n) => (total ? Math.round((n * 100) / total) : 0);
  const maxType = Math.max(1, ...(data?.by_type || []).map((x) => x.count));

  return (
    <>
      <div className="page-head"><h1>Laporan Rekapitulasi</h1></div>
      {error && <div className="alert alert-bad">{error}</div>}

      <div className="rep-grid">
        <div>
          <div className="card">
            <div className="rep-head">
              <Icon name="calendar" size={16} />
              <h2>Rentang Waktu Laporan</h2>
            </div>
            <div className="filters">
              <div className="field" style={{ flex: 1 }}>
                <label>Bulan</label>
                <select value={month} onChange={(e) => setMonth(Number(e.target.value))}>
                  {MONTHS.map((m, i) => <option key={i} value={i}>{m}</option>)}
                </select>
              </div>
              <div className="field" style={{ flex: 1 }}>
                <label>Tahun</label>
                <select value={year} onChange={(e) => setYear(Number(e.target.value))}>
                  {years.map((y) => <option key={y} value={y}>{y}</option>)}
                </select>
              </div>
              <button className="btn btn-primary" onClick={apply}>Terapkan Filter</button>
            </div>
          </div>

          {data && (
            <>
              <div className="card">
                <div className="rep-head">
                  <h2>Ringkasan Operasional</h2>
                  <span className="badge-live">Real-time</span>
                </div>
                <div className="lbl" style={{ display: "flex", alignItems: "center", gap: 6 }}>
                  <Icon name="file" size={14} /> Jenis Layanan
                </div>
                {data.by_type.length === 0 && <p className="muted">Tidak ada data pada periode ini.</p>}
                {data.by_type.map((x) => (
                  <div className="bar-row" key={x.name}>
                    <div className="top"><span>{x.name}</span><b>{x.count}</b></div>
                    <div className="bar-track"><div className="bar-fill" style={{ width: `${(x.count * 100) / maxType}%` }} /></div>
                  </div>
                ))}
              </div>

              <div className="card">
                <div className="rep-head">
                  <Icon name="activity" size={16} />
                  <h2>Status Penyelesaian</h2>
                </div>
                <div className="row" style={{ flexWrap: "nowrap" }}>
                  <div className="pill-stat">
                    <i className="dot-s" style={{ background: "var(--navy)", margin: 0 }} />
                    Selesai <b style={{ marginLeft: "auto" }}>{pct(data.completed)}%</b>
                  </div>
                  <div className="pill-stat alt">
                    <i className="dot-s" style={{ background: "#9db4ff", margin: 0 }} />
                    Diproses <b style={{ marginLeft: "auto" }}>{pct(data.processing)}%</b>
                  </div>
                </div>
                <div className="rep-bar"><div style={{ width: `${pct(data.completed)}%` }} /></div>
                <div className="rep-bar-lbl">
                  <span>Rasio Selesai ({pct(data.completed)}%)</span>
                  <span>Rasio Diproses ({pct(data.processing)}%)</span>
                </div>
                {data.rejected > 0 && (
                  <p className="muted" style={{ marginTop: 10 }}>
                    Ditolak: {data.rejected} pengajuan ({pct(data.rejected)}%)
                  </p>
                )}
              </div>
            </>
          )}
        </div>

        <div className="card">
          <div className="rep-head">
            <Icon name="share" size={16} />
            <h2>Ekspor & Bagikan</h2>
          </div>
          <p className="muted" style={{ marginTop: -4 }}>Unduh rekapitulasi dalam format pilihan.</p>
          <button className="pdf-box" onClick={exportPdf}>
            <Icon name="file" size={20} />
            <span>PDF (.pdf)<small>Format cetak berkas dinas</small></span>
            <Icon name="download" size={16} />
          </button>
          {exportErr && (
            <div className="alert alert-bad" style={{ marginTop: 10 }}>
              <b>⚠ Gagal Unduh</b>
              <div>{exportErr}</div>
            </div>
          )}
        </div>
      </div>
    </>
  );
}