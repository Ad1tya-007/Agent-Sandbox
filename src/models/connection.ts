export type ConnectionState =
  | "connecting"
  | "connected"
  | "disconnected"
  | "error";

export type Connection = {
  state: ConnectionState;
  cluster: string | null;
  message: string | null;
};
