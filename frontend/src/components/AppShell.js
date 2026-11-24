"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { getUser, clearSession, homeFor } from "@/lib/api";

const NAV = {
  mahasiswa: [
    ["/mahasiswa", "Beranda"],
    ["/mahasiswa/pengajuan", "Pengajuan Surat"],
    ["/mahasiswa/status", "Status"],
    ["/mahasiswa/riwayat", "Riwayat"],
    ["/mahasiswa/bantuan", "Bantuan"],
  ],
  staf: [
    ["/staf", "Beranda"],
    ["/staf/antrean", "Antrean Verifikasi"],
    ["/staf/layanan", "Kelola Layanan"],
    ["/staf/laporan", "Laporan"],
    ["/staf/bantuan", "Bantuan"],
    ["/staf/manajemen", "Manajemen"],
  ],
  dosen: [["/dosen", "Beranda"]],
};

export default function AppShell({ role, children }) {
  const router = useRouter();
  const path = usePathname();
  const [user, setUser] = useState(null);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const u = getUser();
    if (!u) return void router.replace("/login");
    if (u.role !== role) return void router.replace(homeFor(u.role));
    setUser(u);
  }, [role, router]);

  if (!user) return <div className="center muted">Memuat…</div>;

  const isActive = (href) => {
    if (href === "/mahasiswa/status" && path.startsWith("/mahasiswa/revisi")) return true;
    if (href === "/dosen" && path.startsWith("/dosen/permohonan")) return true;
    return path === href || (href !== `/${role}` && path.startsWith(href));
  };

  return (
    <>
      <header className="topbar">
        <div className="tb-brand">
          <span className="tb-logo">⌂</span> PORTAL KAMPUS
        </div>
        <nav className="tb-nav">
          {NAV[role].map(([href, label]) => (
            <Link key={href} href={href} className={isActive(href) ? "active" : ""}>
              {label}
            </Link>
          ))}
        </nav>
        <div className="tb-user">
          <button className="avatar" onClick={() => setOpen((v) => !v)} aria-label="Akun">
            ☺
          </button>
          {open && (
            <div className="menu">
              <b>{user.name}</b>
              <div className="muted">{user.identifier}</div>
              <button
                className="btn"
                onClick={() => {
                  clearSession();
                  router.replace("/login");
                }}
              >
                Keluar
              </button>
            </div>
          )}
        </div>
      </header>
      <main className="main">{children}</main>
    </>
  );
}