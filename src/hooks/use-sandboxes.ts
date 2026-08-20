import { useEffect, useState } from "react";
import type { Sandbox } from "@/models/sandbox";
import { connect, subscribeWatch } from "@/websocket/client";

export function useSandboxes(): {
  sandboxes: Sandbox[];
  hydrated: boolean;
} {
  const [sandboxes, setSandboxes] = useState<Sandbox[]>([]);
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => {
    connect();
    return subscribeWatch((event) => {
      switch (event.type) {
        case "snapshot":
          setSandboxes(event.sandboxes);
          setHydrated(true);
          break;
        case "sandbox.added":
          setSandboxes((current) => {
            if (current.some((item) => item.name === event.sandbox.name)) {
              return current.map((item) =>
                item.name === event.sandbox.name ? event.sandbox : item,
              );
            }
            return [...current, event.sandbox];
          });
          break;
        case "sandbox.updated":
          setSandboxes((current) =>
            current.map((item) =>
              item.name === event.sandbox.name ? event.sandbox : item,
            ),
          );
          break;
        case "sandbox.deleted":
          setSandboxes((current) =>
            current.filter((item) => item.name !== event.name),
          );
          break;
      }
    });
  }, []);

  return { sandboxes, hydrated };
}
