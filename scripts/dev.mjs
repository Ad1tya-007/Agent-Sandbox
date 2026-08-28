import { spawn } from "node:child_process";

const backend = spawn("go", ["run", "./cmd/agent-sandbox"], {
  cwd: new URL("../backend", import.meta.url),
  stdio: "inherit",
  env: process.env,
});

const vite = spawn("npx", ["vite"], {
  stdio: "inherit",
  env: process.env,
});

function shutdown(code = 0) {
  backend.kill("SIGTERM");
  vite.kill("SIGTERM");
  process.exit(code);
}

backend.on("exit", (code) => {
  if (code && code !== 0) shutdown(code);
});
vite.on("exit", (code) => {
  if (code && code !== 0) shutdown(code ?? 0);
});

process.on("SIGINT", () => shutdown(0));
process.on("SIGTERM", () => shutdown(0));
