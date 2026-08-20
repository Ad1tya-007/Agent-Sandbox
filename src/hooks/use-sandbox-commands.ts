import { useCallback, useState } from "react";
import { toast } from "sonner";
import type { CreateSandboxInput } from "@/models/sandbox";
import { errorMessage } from "@/services/errors";
import {
  createSandbox,
  deleteSandbox,
  pauseSandbox,
  resumeSandbox,
} from "@/services/sandboxes";

export type SandboxAction = "create" | "pause" | "resume" | "delete";

export function useSandboxCommands() {
  const [pending, setPending] = useState<Record<string, SandboxAction>>({});

  const track = useCallback(async function track<T>(
    name: string,
    action: SandboxAction,
    run: () => Promise<T>,
  ): Promise<T> {
    setPending((current) => ({ ...current, [name]: action }));
    try {
      return await run();
    } catch (err) {
      toast.error(errorMessage(err));
      throw err;
    } finally {
      setPending((current) => {
        const next = { ...current };
        delete next[name];
        return next;
      });
    }
  }, []);

  const create = useCallback(
    async (input: CreateSandboxInput) => {
      const result = await track(input.name, "create", () =>
        createSandbox(input),
      );
      toast.success(`${input.name} created`);
      return result;
    },
    [track],
  );

  const pause = useCallback(
    async (name: string) => {
      await track(name, "pause", () => pauseSandbox(name));
      toast.success(`${name} paused`);
    },
    [track],
  );

  const resume = useCallback(
    async (name: string) => {
      await track(name, "resume", () => resumeSandbox(name));
      toast.success(`${name} resumed`);
    },
    [track],
  );

  const remove = useCallback(
    async (name: string) => {
      await track(name, "delete", () => deleteSandbox(name));
      toast.success(`Deleting ${name}`);
    },
    [track],
  );

  return { create, pause, resume, remove, pending };
}
