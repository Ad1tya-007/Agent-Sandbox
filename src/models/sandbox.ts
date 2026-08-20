export type SandboxPhase =
  | "Pending"
  | "Running"
  | "Paused"
  | "Failed"
  | "Terminating";

export type Condition = {
  type: string;
  status: "True" | "False" | "Unknown";
  message: string;
};

export type TimelineEvent = {
  id: string;
  title: string;
  detail: string;
  at: string;
};

export type Sandbox = {
  name: string;
  namespace: string;
  status: SandboxPhase;
  image: string;
  cpu: string;
  memory: string;
  node: string | null;
  ip: string | null;
  createdAt: string;
  persistentStorage: boolean;
  conditions: Condition[];
  events: TimelineEvent[];
  yaml: string;
};

export type CreateSandboxInput = {
  name: string;
  image: string;
  cpu: string;
  memory: string;
  persistentStorage: boolean;
};

export type LogLine = {
  id: string;
  ts: string;
  message: string;
};
