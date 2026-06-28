export type TemplateConfigField = {
  name: string;
  label: string;
  type: "string" | "text" | "color" | "select" | string;
  required: boolean;
  default: string;
  placeholder?: string;
  help?: string;
  options?: string[];
};

export type ProjectTemplate = {
  id: string;
  slug: string;
  name: string;
  category: string;
  categoryLabel: string;
  description: string;
  summary: string;
  tags: string[];
  authorUserId?: string;
  authorName: string;
  source: "official" | "community" | string;
  interactiveRequired: boolean;
  analyticsRecommended: boolean;
  usageCount: number;
  configFields: TemplateConfigField[];
  collections: Array<{
    name: string;
    permissions: { publicRead: boolean; publicWrite: boolean };
    fields: Array<{ name: string; type: string; required: boolean; isList: boolean }>;
  }>;
  createdAtText: string;
};

export type TemplateCategory = {
  id: string;
  label: string;
};

export type TemplateSubmission = {
  id: string;
  authorUserId: string;
  authorEmail?: string;
  authorName: string;
  slug: string;
  name: string;
  category: string;
  categoryLabel: string;
  summary: string;
  description: string;
  tags: string[];
  interactiveRequired: boolean;
  analyticsRecommended: boolean;
  configFields: TemplateConfigField[];
  collections: ProjectTemplate["collections"];
  sourceType: string;
  status: "pending" | "published" | "rejected" | string;
  adminNote: string;
  reviewedBy: string;
  reviewedAt: string;
  createdAt: string;
  updatedAt: string;
};
