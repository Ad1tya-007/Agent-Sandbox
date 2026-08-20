import { useState, type FormEvent } from "react";
import type { CreateSandboxInput } from "@/models/sandbox";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";

const EMPTY: CreateSandboxInput = {
  name: "",
  image: "",
  cpu: "500m",
  memory: "1Gi",
  persistentStorage: false,
};

export function CreateSandboxDialog({
  open,
  onOpenChange,
  pending,
  onCreate,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  pending: boolean;
  onCreate: (input: CreateSandboxInput) => Promise<void>;
}) {
  const [form, setForm] = useState<CreateSandboxInput>(EMPTY);

  function update<K extends keyof CreateSandboxInput>(
    key: K,
    value: CreateSandboxInput[K],
  ) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    await onCreate(form);
    setForm(EMPTY);
    onOpenChange(false);
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (pending) return;
        if (!next) setForm(EMPTY);
        onOpenChange(next);
      }}
    >
      <DialogContent className="sm:max-w-md" showCloseButton={!pending}>
        <DialogHeader>
          <DialogTitle>Create sandbox</DialogTitle>
          <DialogDescription>
            The backend will validate this spec and apply it to the cluster. The
            list updates when the watch event arrives.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="grid gap-4">
          <FieldGroup className="gap-3">
            <Field>
              <FieldLabel htmlFor="sandbox-name">Name</FieldLabel>
              <Input
                id="sandbox-name"
                autoFocus
                required
                autoComplete="off"
                placeholder="research-agent"
                value={form.name}
                onChange={(e) => update("name", e.target.value)}
              />
              <FieldDescription>
                DNS label. Lowercase letters, numbers, hyphens.
              </FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor="sandbox-image">Container image</FieldLabel>
              <Input
                id="sandbox-image"
                required
                autoComplete="off"
                placeholder="python:3.12-slim"
                value={form.image}
                onChange={(e) => update("image", e.target.value)}
              />
            </Field>
            <div className="grid grid-cols-2 gap-3">
              <Field>
                <FieldLabel htmlFor="sandbox-cpu">CPU</FieldLabel>
                <Input
                  id="sandbox-cpu"
                  required
                  autoComplete="off"
                  placeholder="500m"
                  value={form.cpu}
                  onChange={(e) => update("cpu", e.target.value)}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="sandbox-memory">Memory</FieldLabel>
                <Input
                  id="sandbox-memory"
                  required
                  autoComplete="off"
                  placeholder="1Gi"
                  value={form.memory}
                  onChange={(e) => update("memory", e.target.value)}
                />
              </Field>
            </div>
            <Field orientation="horizontal">
              <Checkbox
                id="sandbox-pvc"
                checked={form.persistentStorage}
                onCheckedChange={(checked) =>
                  update("persistentStorage", checked === true)
                }
              />
              <FieldLabel htmlFor="sandbox-pvc" className="font-normal">
                Persistent storage
                <FieldDescription>
                  Attach a workspace PersistentVolumeClaim.
                </FieldDescription>
              </FieldLabel>
            </Field>
          </FieldGroup>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={pending}
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? <Spinner /> : null}
              Create
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
