package http

import (
	"fmt"
	"log"
	"strings"

	"ai-static-host/api/internal/domain"
)

func (rt *Router) notifyWelcomeUser(user domain.User) {
	if rt.mailer == nil || strings.TrimSpace(user.Email) == "" {
		return
	}
	name := strings.TrimSpace(user.Username)
	if name == "" {
		name = "新朋友"
	}
	subject := "欢迎来到 PlayPage"
	body := fmt.Sprintf("%s，你好，\n\n欢迎来到 PlayPage！\n\n你可以在这里上传 HTML 作品，把自己的网页、小游戏、工具或故事页面发布出去。\n\n你可以先尝试这几件事：\n1. 在作品管理页面上传第一个作品。\n2. 如果作品需要留言、云存档、排行榜等功能，可以开启互动功能。\n3. 如果 AI 写出的页面跑不通，可以在作品管理页面提交急救站申请。\n4. 如果需要更独立的访问地址，可以为作品申请独立域名。\n\n祝你玩得开心，也欢迎把自己的作品分享给朋友。\n\nPlayPage", name)
	go func() {
		if err := rt.mailer.SendText(user.Email, subject, body); err != nil {
			log.Printf("send welcome mail failed: user_id=%s err=%v", user.ID, err)
		}
	}()
}

func shouldSendWelcomeAfterProfileUpdate(before, after domain.User) bool {
	beforeName := strings.TrimSpace(before.Username)
	afterName := strings.TrimSpace(after.Username)
	if afterName == "" || beforeName == afterName {
		return false
	}
	emailPrefix := strings.Split(strings.TrimSpace(before.Email), "@")[0]
	return beforeName == "" || beforeName == emailPrefix
}

func (rt *Router) notifyProjectDomainReview(item domain.ProjectDomain) {
	if rt.mailer == nil || strings.TrimSpace(item.OwnerEmail) == "" {
		return
	}
	subject := "PlayPage 独立域名申请结果"
	statusText := projectDomainStatusText(item.Status)
	body := fmt.Sprintf("你好，\n\n你的作品《%s》的独立域名申请已有结果。\n\n申请域名：%s\n当前状态：%s\n", item.ProjectName, item.Domain, statusText)
	if strings.TrimSpace(item.RejectReason) != "" {
		body += "原因说明：" + strings.TrimSpace(item.RejectReason) + "\n"
	}
	if strings.TrimSpace(item.AdminNote) != "" {
		body += "补充说明：" + strings.TrimSpace(item.AdminNote) + "\n"
	}
	body += "\n你可以登录 PlayPage，在作品管理页面查看详情。\n\nPlayPage"
	go func() {
		if err := rt.mailer.SendText(item.OwnerEmail, subject, body); err != nil {
			log.Printf("send project domain review mail failed: domain_id=%s err=%v", item.ID, err)
		}
	}()
}

func (rt *Router) notifyRepairRequestStatus(item domain.RepairRequest) {
	if rt.mailer == nil || strings.TrimSpace(item.OwnerEmail) == "" {
		return
	}
	subject := "PlayPage 急救站申请状态更新"
	statusText := repairStatusText(item.Status)
	body := fmt.Sprintf("你好，\n\n你的作品《%s》的急救站申请状态已更新。\n\n当前状态：%s\n", item.ProjectName, statusText)
	if strings.TrimSpace(item.AdminReply) != "" {
		body += "管理员回复：" + strings.TrimSpace(item.AdminReply) + "\n"
	}
	if item.Status == "need_info" {
		body += "\n请登录 PlayPage，在急救站申请页面补充一次所需信息。\n"
	} else {
		body += "\n你可以登录 PlayPage，在作品管理页面查看详情。\n"
	}
	body += "\nPlayPage"
	go func() {
		if err := rt.mailer.SendText(item.OwnerEmail, subject, body); err != nil {
			log.Printf("send repair request status mail failed: request_id=%s err=%v", item.ID, err)
		}
	}()
}

func projectDomainStatusText(status string) string {
	switch status {
	case "active":
		return "已通过"
	case "rejected":
		return "未通过"
	case "disabled":
		return "已停用"
	case "pending":
		return "待审核"
	default:
		return status
	}
}

func repairStatusText(status string) string {
	switch status {
	case "pending":
		return "待处理"
	case "processing":
		return "处理中"
	case "need_info":
		return "需要补充信息"
	case "fixed":
		return "已修复"
	case "rejected":
		return "无法处理"
	case "closed":
		return "已关闭"
	default:
		return status
	}
}
