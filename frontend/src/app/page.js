"use client";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { getUser, homeFor } from "@/lib/api";

export default function Root() {
  const router = useRouter();
  useEffect(() => {
    const u = getUser();
    router.replace(u ? homeFor(u.role) : "/login");
  }, [router]);
  return <div className="center muted">Memuat…</div>;
}