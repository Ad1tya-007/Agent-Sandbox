import type { Sandbox } from "@/models/sandbox";

export function sandboxToYaml(sandbox: Sandbox): string {
  const storage = sandbox.persistentStorage
    ? `  persistence:
    enabled: true
    size: 10Gi
`
    : `  persistence:
    enabled: false
`;

  const conditions = sandbox.conditions
    .map(
      (condition) => `    - type: ${condition.type}
      status: "${condition.status}"
      message: ${quote(condition.message)}`,
    )
    .join("\n");

  return `apiVersion: agents.x-k8s.io/v1alpha1
kind: Sandbox
metadata:
  name: ${sandbox.name}
  namespace: ${sandbox.namespace}
  creationTimestamp: "${sandbox.createdAt}"
  labels:
    app.kubernetes.io/managed-by: agent-sandbox
spec:
  template:
    spec:
      containers:
        - name: sandbox
          image: ${sandbox.image}
          resources:
            requests:
              cpu: "${sandbox.cpu}"
              memory: "${sandbox.memory}"
${storage}status:
  phase: ${sandbox.status}
  podIP: ${sandbox.ip ?? "null"}
  host: ${sandbox.node ?? "null"}
  conditions:
${conditions}
`;
}

function quote(value: string): string {
  if (/[:#]/.test(value) || value.includes(" ")) {
    return JSON.stringify(value);
  }
  return value;
}
