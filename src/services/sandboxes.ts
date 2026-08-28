import { mockBackend } from "@/mock/backend";
import type { CreateSandboxInput } from "@/models/sandbox";
import { ApiError } from "@/services/errors";

const useMock = import.meta.env.VITE_USE_MOCK === "true";
const apiBase = (
  import.meta.env.VITE_API_BASE ?? "http://127.0.0.1:8787"
).replace(/\/$/, "");

async function parseError(response: Response): Promise<ApiError> {
  let message = response.statusText || "Request failed";
  try {
    const body = (await response.json()) as { error?: unknown };
    if (typeof body.error === "string" && body.error.trim()) {
      message = body.error;
    }
  } catch {
    // Keep statusText when the body is empty or not JSON.
  }
  return new ApiError(message, response.status);
}

async function request(path: string, init: RequestInit): Promise<Response> {
  let response: Response;
  try {
    response = await fetch(`${apiBase}${path}`, init);
  } catch {
    throw new ApiError("Cannot reach the Agent Sandbox backend.", 0);
  }
  if (!response.ok) {
    throw await parseError(response);
  }
  return response;
}

export async function createSandbox(
  input: CreateSandboxInput,
): Promise<{ name: string }> {
  if (useMock) return mockBackend.create(input);
  const response = await request("/api/sandboxes", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  return (await response.json()) as { name: string };
}

export async function pauseSandbox(name: string): Promise<void> {
  if (useMock) return mockBackend.pause(name);
  await request(`/api/sandboxes/${encodeURIComponent(name)}/pause`, {
    method: "POST",
  });
}

export async function resumeSandbox(name: string): Promise<void> {
  if (useMock) return mockBackend.resume(name);
  await request(`/api/sandboxes/${encodeURIComponent(name)}/resume`, {
    method: "POST",
  });
}

export async function deleteSandbox(name: string): Promise<void> {
  if (useMock) return mockBackend.remove(name);
  await request(`/api/sandboxes/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });
}
