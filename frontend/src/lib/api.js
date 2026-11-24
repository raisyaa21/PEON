const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export const STATUS_LABEL = {
  submitted: "Pengajuan Diterima",
  awaiting_recommendation: "Menunggu Rekomendasi Dosen PA",
  reviewing: "Pemeriksaan Berkas",
  revision: "Perlu Revisi",
  approving: "Pengesahan Surat",
  completed: "Selesai Diterbitkan",
  rejected: "Ditolak",
};

export const MAX_SIZE = 5 * 1024 * 1024;

export function homeFor(role) {
  return { mahasiswa: "/mahasiswa", dosen: "/dosen", staf: "/staf" }[role] || "/login";
}

export function getToken() {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("token");
}

export function getUser() {
  if (typeof window === "undefined") return null;
  try {
    return JSON.parse(localStorage.getItem("user") || "null");
  } catch {
    return null;
  }
}

export function saveSession(token, user) {
  localStorage.setItem("token", token);
  localStorage.setItem("user", JSON.stringify(user));
}

export function clearSession() {
  localStorage.removeItem("token");
  localStorage.removeItem("user");
}

export async function api(path, { method = "GET", body, form } = {}) {
  const headers = {};
  const token = getToken();
  if (token) headers.Authorization = `Bearer ${token}`;

  let payload;
  if (form) {
    payload = form;
  } else if (body) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }

  const res = await fetch(API + path, { method, headers, body: payload });
  if (res.status === 401 && token) {
    clearSession();
    window.location.href = "/login";
    throw new Error("Sesi berakhir, silakan login kembali");
  }
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || "Terjadi kesalahan");
  return data;
}

// file butuh header Authorization, jadi diambil sebagai blob dulu
export async function fetchBlob(path) {
  const res = await fetch(API + path, { headers: { Authorization: `Bearer ${getToken()}` } });
  if (!res.ok) throw new Error("Dokumen tidak dapat dibuka saat ini.");
  return res.blob();
}

export async function openPath(path) {
  const blob = await fetchBlob(path);
  window.open(URL.createObjectURL(blob), "_blank");
}

export async function downloadPath(path, filename) {
  const blob = await fetchBlob(path);
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1500);
}

export const openDocument = (id) => openPath(`/api/documents/${id}/file`);

export function validateFile(file) {
  const okExt = /\.(pdf|jpe?g)$/i.test(file.name);
  const okType = ["application/pdf", "image/jpeg"].includes(file.type);
  if (!okExt || !okType) return "Format file harus PDF atau JPG.";
  if (file.size > MAX_SIZE)
    return `Ukuran file ${(file.size / 1048576).toFixed(1)} MB melebihi batas 5 MB.`;
  return "";
}

// status rekomendasi Dosen PA -> [label, kelas css]
export function advisorState(t) {
  if (t.status === "awaiting_recommendation") return ["Menunggu", "wait"];
  if (t.advisor_status === "rejected") return ["Ditolak", "no"];
  return ["Direkomendasikan", "ok"];
}

export const fmt = (d) =>
  new Date(d).toLocaleString("id-ID", { dateStyle: "medium", timeStyle: "short" });

export const fmtDay = (d) => new Date(d).toLocaleDateString("id-ID", { dateStyle: "long" });

export function elapsed(from) {
  const m = Math.max(0, Math.floor((Date.now() - new Date(from).getTime()) / 60000));
  if (m < 1) return "kurang dari 1 menit";
  if (m < 60) return `${m} menit`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h} jam ${m % 60} menit`;
  return `${Math.floor(h / 24)} hari ${h % 24} jam`;
}