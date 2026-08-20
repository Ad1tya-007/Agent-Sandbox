export type ResourceKind =
  | "Sandbox"
  | "Pod"
  | "PersistentVolumeClaim"
  | "Service"
  | "Events";

export type ResourceNode = {
  id: string;
  kind: ResourceKind;
  name: string;
  children: ResourceNode[];
};

export type ResourceField = {
  label: string;
  value: string;
};

export type ResourceDetail = {
  id: string;
  kind: ResourceKind;
  name: string;
  fields: ResourceField[];
};
