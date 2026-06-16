export type Project = { id: string; name: string; publicUrl: string; currentReleaseId?: string };
export type ProjectResponse = { project: Project };
export type RepairRequest = {
  id: string;
  projectId: string;
  issueType: string;
  description: string;
  expected: string;
  allowAdminEdit: boolean;
  contact: string;
  status: string;
  adminReply: string;
  userReply: string;
  userRepliedAt: string;
  createdAt: string;
  updatedAt: string;
};
export type RepairAIJob = {
  id: string;
  repairRequestId: string;
  projectId: string;
  status: string;
  round: number;
  feedback: string;
  previewUrl: string;
  errorMessage: string;
  createdAt: string;
  updatedAt: string;
  finishedAt: string;
};
export type RepairAIMessage = {
  id: string;
  jobId: string;
  agentKey: string;
  agentName: string;
  role: string;
  messageType: string;
  content: string;
  messageSeq: number;
  createdAt: string;
};
export type StatusTone = "info" | "success" | "error";

export const issueOptions = [
  ["page_broken", "页面打不开或白屏"],
  ["button_broken", "按钮没反应"],
  ["interactive_error", "互动功能失败"],
  ["data_error", "数据表或存档问题"],
  ["encoding_error", "中文乱码"],
  ["style_error", "样式错乱"],
  ["ai_code_error", "AI 生成代码跑不通"],
  ["other", "其他问题"]
] as const;

export function issueLabel(value: string): string {
  return issueOptions.find(([key]) => key === value)?.[1] ?? value;
}

export function requestStatusLabel(status: string): string {
  switch (status) {
    case "pending": return "待管理员处理";
    case "processing": return "管理员处理中";
    case "need_info": return "管理员需要你补充信息";
    case "fixed": return "管理员已修复";
    case "rejected": return "管理员无法处理";
    case "closed": return "已关闭";
    default: return status || "未知状态";
  }
}

export function aiStatusLabel(status: string): string {
  switch (status) {
    case "running": return "AI 圆桌正在修";
    case "ready_preview": return "AI 已生成修复版，等待预览";
    case "published": return "AI 已修复并发布";
    case "needs_admin": return "AI 没修好，已转管理员";
    case "failed": return "AI 圆桌失败";
    case "canceled": return "AI 圆桌已叫停";
    default: return "还没有启动 AI 圆桌";
  }
}

export function combinedRepairStatus(item: RepairRequest, job?: RepairAIJob | null): string {
  if (job?.status === "published") return "AI 已修复并发布";
  if (job?.status === "ready_preview") return "AI 已生成修复版，等待确认";
  if (job?.status === "running") return "AI 圆桌正在修";
  if (job?.status === "canceled") return "AI 圆桌已叫停";
  if (job?.status === "needs_admin") return "AI 没修好，已转管理员";
  if (job?.status === "failed") return "AI 圆桌失败，等待管理员处理";
  return requestStatusLabel(item.status);
}

export function isFinalAIStatus(status: string): boolean {
  return ["ready_preview", "published", "failed", "needs_admin", "canceled"].includes(status);
}
