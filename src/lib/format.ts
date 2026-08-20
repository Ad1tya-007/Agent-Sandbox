import { format } from "date-fns";

export function formatAge(iso: string, now = Date.now()): string {
  const ms = Math.max(0, now - new Date(iso).getTime());
  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h`;
  const days = Math.floor(hours / 24);
  return `${days}d`;
}

export function formatTimestamp(iso: string): string {
  return format(new Date(iso), "MMM d, HH:mm");
}

export function formatClock(iso: string): string {
  return format(new Date(iso), "HH:mm:ss");
}
