import type { TimelineEvent } from "@/models/sandbox";
import { formatAge, formatTimestamp } from "@/lib/format";

export function EventsTab({
  events,
  now,
}: {
  events: TimelineEvent[];
  now: number;
}) {
  if (events.length === 0) {
    return (
      <p className="text-muted-foreground p-4 text-sm">
        No lifecycle events yet.
      </p>
    );
  }

  return (
    <ol className="p-4">
      {events.map((event, index) => (
        <li key={event.id} className="flex gap-3">
          <div className="flex w-3 shrink-0 flex-col items-center self-stretch">
            <span className="bg-foreground mt-1.5 size-2 shrink-0 rounded-full" />
            {index < events.length - 1 ? (
              <span className="bg-muted-foreground/40 mt-1 w-px flex-1" />
            ) : null}
          </div>
          <div className={index < events.length - 1 ? "pb-5" : "pb-0"}>
            <div className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
              <p className="text-sm font-medium">{event.title}</p>
              <time
                className="text-muted-foreground text-xs tabular-nums"
                dateTime={event.at}
                title={formatTimestamp(event.at)}
              >
                {formatAge(event.at, now)} ago
              </time>
            </div>
            <p className="text-muted-foreground mt-0.5 text-xs">
              {event.detail}
            </p>
          </div>
        </li>
      ))}
    </ol>
  );
}
