"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { getUser, clearSession } from "@/lib/api";
import Icon from "@/components/Icon";

const MENU = [
  ["/staf/antrean", "Antrean Verifikasi", "file"],
  ["/staf/layanan", "Kelola Layanan & Template", "layers"],
  ["/staf/bantuan", "Pesan Bantuan", "chat"],
  ["/staf/laporan", "Laporan Rekapitulasi", "chart"],
  ["/staf/manajemen", "Manajemen Hak Akses", "users"],
];

const initials = (n = "") =>
  n.replace(/[.,]/g, " ").split(" ").filter((w) => /^[A-Z]/.test(w) && w.length > 2).slice(0, 2).map((w) => w[0]).join("") || "S";

export default function StafHome() {
  const router = useRouter();
  const [user, setUser] = useState(null);
  useEffect(() => setUser(getUser()), []);

  return (
    <>
      <div className="hero-dark">
        <span className="avatar sq">{initials(user?.name)}</span>
        <div style={{ flex: 1 }}>
          <b>{user?.name}</b>
          <div style={{ opacity: 0.7, fontSize: ".8rem" }}>Staf Layanan Akademik • Kampus Pusat</div>
        </div>
        <span className="chip live">● Sistem Aktif</span>
      </div>

      <h1 style={{ marginBottom: 12 }}>Menu Utama Staf</h1>
      <div className="menu-list">
        {MENU.map(([href, label, icon]) => (
          <Link key={href} href={href}>
            <span className="mi"><Icon name={icon} /></span>
            <span className="lbl-t">{label}</span>
            <span className="chev"><Icon name="chevron" /></span>
          </Link>
        ))}
      </div>

      <button
        className="btn btn-danger btn-block btn-ico"
        style={{ padding: 12 }}
        onClick={() => {
          clearSession();
          router.replace("/login");
        }}
      >
        Keluar Akun Staf <Icon name="logout" size={16} />
      </button>
    </>
  );
}