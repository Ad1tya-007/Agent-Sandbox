import { useEffect, useState } from "react";
import type { Connection } from "@/models/connection";
import { connect, subscribeWatch } from "@/websocket/client";

const INITIAL: Connection = {
  state: "connecting",
  cluster: null,
  message: "Connecting",
};

export function useConnection(): Connection {
  const [connection, setConnection] = useState<Connection>(INITIAL);

  useEffect(() => {
    connect();
    return subscribeWatch((event) => {
      if (event.type === "connection") setConnection(event.connection);
    });
  }, []);

  return connection;
}
