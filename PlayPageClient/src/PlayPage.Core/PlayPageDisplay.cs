namespace PlayPage.Core;

public static class PlayPageDisplay
{
    public static string YesNo(bool value) => value ? "已开启" : "未开启";

    public static string Visibility(string value) => value switch
    {
        "public" => "已显示到广场",
        "unlisted" => "不在广场显示",
        "private" => "仅自己可见",
        _ => string.IsNullOrWhiteSpace(value) ? "未知" : value
    };

    public static string Plan(string value) => value switch
    {
        "light" => "轻享版",
        "support" => "支持版",
        "admin" => "最高管理员",
        "free" or "" => "免费版",
        _ => value
    };

    public static string Role(string value) => value switch
    {
        "admin" => "管理员",
        "super_admin" => "最高管理员",
        "user" or "" => "普通用户",
        _ => value
    };

    public static string PaymentMethod(string value) => value switch
    {
        "wechat" => "微信支付",
        "alipay" => "支付宝",
        _ => string.IsNullOrWhiteSpace(value) ? "未填写" : value
    };

    public static string Status(string value) => value switch
    {
        "pending" => "待处理",
        "approved" => "已通过",
        "rejected" => "已拒绝",
        "active" => "已生效",
        "completed" => "已完成",
        "fixed" => "管理员已修复",
        "ai_fixed" => "AI 已修复",
        "need_info" => "需要补充信息",
        "closed" => "已关闭",
        "running" => "正在处理",
        "succeeded" => "已完成",
        "failed" => "失败",
        "canceled" => "已取消",
        "deleted" => "已删除",
        _ => string.IsNullOrWhiteSpace(value) ? "未知" : value
    };

    public static string AiStatus(string value) => value switch
    {
        "running" => "AI 圆桌正在修",
        "succeeded" => "AI 圆桌已生成修复版本",
        "published" => "AI 修复已发布",
        "failed" => "AI 圆桌失败，等待管理员处理",
        "canceled" => "AI 圆桌已叫停",
        "" => "还没有启动 AI 圆桌",
        _ => Status(value)
    };

    public static string IssueType(string value) => value switch
    {
        "page_broken" => "页面打不开或白屏",
        "button_broken" => "按钮没反应",
        "interactive_api" => "互动功能报错",
        "encoding" => "中文乱码",
        "style" => "样式显示异常",
        "other" or "" => "其他问题",
        _ => value
    };
}
