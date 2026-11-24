"use client";
import ChatBox from "@/components/ChatBox";

export default function Bantuan() {
  return (
    <div className="chat-shell">
      <div className="chat-head">
        <span className="avatar">▤</span>
        <div>
          <b>Staf Layanan Akademik</b>
          <small>● Biro Administrasi Akademik • Online</small>
        </div>
      </div>
      <ChatBox role="mahasiswa" />
    </div>
  );
}