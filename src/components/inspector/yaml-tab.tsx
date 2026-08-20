import { Copy } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";

export function YamlTab({ yaml }: { yaml: string }) {
  async function copy() {
    await navigator.clipboard.writeText(yaml);
    toast.success("YAML copied");
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex shrink-0 justify-end border-b px-3 py-2">
        <Button type="button" variant="ghost" size="xs" onClick={copy}>
          <Copy />
          Copy
        </Button>
      </div>
      <pre className="min-h-0 flex-1 overflow-auto p-3 font-mono text-[12px] leading-5">
        {yaml}
      </pre>
    </div>
  );
}
