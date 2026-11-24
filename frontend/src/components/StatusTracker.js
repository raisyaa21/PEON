import { fmt, elapsed } from "@/lib/api";

const STAGES = [
  ["submitted", "Pengajuan Diterima"],
  ["reviewing", "Pemeriksaan Berkas"],
  ["approving", "Pengesahan Surat"],
  ["completed", "Surat Selesai Diterbitkan"],
];

const idxOf = (s) =>
  s === "revision" ? 1 : s === "awaiting_recommendation" ? 0 : STAGES.findIndex(([k]) => k === s);

export default function StatusTracker({ ticket }) {
  const reached = Math.max(0, ...ticket.history.map((h) => idxOf(h.status)));
  const finished = ticket.status === "completed";
  const rejected = ticket.status === "rejected";

  return (
    <div className="htrack">
      {STAGES.map(([key, label], i) => {
        const entries = ticket.history.filter((h) => {
          if (h.status === key) return true;
          if (key === "submitted" && h.status === "awaiting_recommendation") return true;
          if (key === "reviewing" && h.status === "revision") return true;
          if (rejected && i === reached && h.status === "rejected") return true;
          return false;
        });
        const last = entries[entries.length - 1];

        let state = "pending";
        if (i < reached || (finished && i === reached)) state = "done";
        else if (i === reached) state = rejected ? "stopped" : "active";

        return (
          <div key={key} className={`hstep ${state}`}>
            <div className="bar" />
            <b>{label}</b>
            <small>
              {last ? (
                <>
                  {fmt(last.created_at)} · {last.unit}
                  {state === "active" && <> · berjalan {elapsed(ticket.updated_at)}</>}
                </>
              ) : (
                "Menunggu"
              )}
            </small>
          </div>
        );
      })}
    </div>
  );
}