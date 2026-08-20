import type { ResourceDetail, ResourceNode } from "@/models/resource";
import type { Sandbox } from "@/models/sandbox";

export function resourceTreeFor(sandbox: Sandbox): ResourceNode {
  const children: ResourceNode[] = [
    {
      id: `${sandbox.name}:pod`,
      kind: "Pod",
      name: `${sandbox.name}-0`,
      children: [],
    },
  ];

  if (sandbox.persistentStorage) {
    children.push({
      id: `${sandbox.name}:pvc`,
      kind: "PersistentVolumeClaim",
      name: `${sandbox.name}-workspace`,
      children: [],
    });
  }

  children.push(
    {
      id: `${sandbox.name}:svc`,
      kind: "Service",
      name: sandbox.name,
      children: [],
    },
    {
      id: `${sandbox.name}:events`,
      kind: "Events",
      name: "Events",
      children: [],
    },
  );

  return {
    id: `${sandbox.name}:sandbox`,
    kind: "Sandbox",
    name: sandbox.name,
    children,
  };
}

export function resourceDetailFor(
  sandbox: Sandbox,
  node: ResourceNode,
): ResourceDetail {
  switch (node.kind) {
    case "Sandbox":
      return {
        id: node.id,
        kind: node.kind,
        name: node.name,
        fields: [
          { label: "Kind", value: "Sandbox" },
          { label: "Name", value: sandbox.name },
          { label: "Namespace", value: sandbox.namespace },
          { label: "Phase", value: sandbox.status },
          { label: "Image", value: sandbox.image },
        ],
      };
    case "Pod":
      return {
        id: node.id,
        kind: node.kind,
        name: node.name,
        fields: [
          { label: "Kind", value: "Pod" },
          { label: "Name", value: node.name },
          { label: "Namespace", value: sandbox.namespace },
          { label: "Node", value: sandbox.node ?? "—" },
          { label: "Pod IP", value: sandbox.ip ?? "—" },
          { label: "Owner", value: `Sandbox/${sandbox.name}` },
        ],
      };
    case "PersistentVolumeClaim":
      return {
        id: node.id,
        kind: node.kind,
        name: node.name,
        fields: [
          { label: "Kind", value: "PersistentVolumeClaim" },
          { label: "Name", value: node.name },
          { label: "Namespace", value: sandbox.namespace },
          { label: "Access", value: "ReadWriteOnce" },
          { label: "Request", value: "10Gi" },
          { label: "Owner", value: `Sandbox/${sandbox.name}` },
        ],
      };
    case "Service":
      return {
        id: node.id,
        kind: node.kind,
        name: node.name,
        fields: [
          { label: "Kind", value: "Service" },
          { label: "Name", value: node.name },
          { label: "Namespace", value: sandbox.namespace },
          { label: "Type", value: "ClusterIP" },
          { label: "Selector", value: `sandbox=${sandbox.name}` },
          { label: "Owner", value: `Sandbox/${sandbox.name}` },
        ],
      };
    case "Events":
      return {
        id: node.id,
        kind: node.kind,
        name: "Events",
        fields: [
          { label: "Kind", value: "Events" },
          { label: "Sandbox", value: sandbox.name },
          { label: "Count", value: String(sandbox.events.length) },
          {
            label: "Latest",
            value: sandbox.events[sandbox.events.length - 1]?.title ?? "—",
          },
        ],
      };
  }
}
