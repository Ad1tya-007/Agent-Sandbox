import { mockBackend } from "@/mock/backend";
import type { CreateSandboxInput } from "@/models/sandbox";

export async function createSandbox(input: CreateSandboxInput) {
  return mockBackend.create(input);
}

export async function pauseSandbox(name: string) {
  return mockBackend.pause(name);
}

export async function resumeSandbox(name: string) {
  return mockBackend.resume(name);
}

export async function deleteSandbox(name: string) {
  return mockBackend.remove(name);
}
