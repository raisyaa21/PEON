import { STATUS_LABEL } from "@/lib/api";

export default function StatusBadge({ status }) {
  return <span className={`badge ${status}`}>{STATUS_LABEL[status] || status}</span>;
}