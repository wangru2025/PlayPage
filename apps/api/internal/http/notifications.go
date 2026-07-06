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

func (rt *Router) notifyContestSubmissionReview(item domain.ContestSubmission) {
	if rt.mailer == nil || strings.TrimSpace(item.UserEmail) == "" {
		return
	}
	status := strings.TrimSpace(item.Status)
	if status != "shortlisted" && status != "winner" && status != "rejected" {
		return
	}

	projectName := strings.TrimSpace(item.ProjectName)
	if projectName == "" {
		projectName = "你的作品"
	}
	note := strings.TrimSpace(item.AdminNote)
	var subject string
	var body string
	switch status {
	case "shortlisted":
		subject = "PlayPage 创作比赛入围通知"
		body = fmt.Sprintf("你好，\n\n恭喜！你的作品《%s》已入围 PlayPage 作品创作比赛。\n\n", projectName)
		if note != "" {
			body += "管理员备注：" + note + "\n\n"
		}
		body += "后续评选结果会在活动结束后继续通知。\n\nPlayPage"
	case "winner":
		subject = "PlayPage 创作比赛获奖通知"
		body = fmt.Sprintf("你好，\n\n恭喜！你的作品《%s》已在 PlayPage 作品创作比赛中获奖。\n\n领奖方式和后续安排会另行通知，请留意你的邮箱或平台消息。\n", projectName)
		if note != "" {
			body += "\n管理员备注：" + note + "\n"
		}
		body += "\nPlayPage"
	case "rejected":
		subject = "PlayPage 创作比赛评选结果通知"
		body = fmt.Sprintf("你好，\n\n你的作品《%s》本次没有入选 PlayPage 作品创作比赛。\n", projectName)
		if note != "" {
			body += "\n原因说明：" + note + "\n"
		}
		body += "\n这不影响作品继续在 PlayPage 发布和分享，也欢迎你后续继续投稿新作品。\n\nPlayPage"
	}

	go func() {
		if err := rt.mailer.SendText(item.UserEmail, subject, body); err != nil {
			log.Printf("send contest submission review mail failed: submission_id=%s err=%v", item.ID, err)
		}
	}()
}

func (rt *Router) notifyProjectProposalCreated(item domain.ProjectProposal) {
	if rt.mailer == nil || strings.TrimSpace(item.TargetOwnerEmail) == "" {
		return
	}
	subject := "PlayPage 收到新的改进提案"
	body := fmt.Sprintf("你好，\n\n你的作品《%s》收到了来自 @%s 的改进提案。\n\n提案标题：%s\n提案说明：\n%s\n\n你可以登录 PlayPage，在作品的改进提案页面查看并决定是否采纳。\n\nPlayPage",
		item.TargetProjectName, item.AuthorUsername, item.Title, item.Body)
	go func() {
		if err := rt.mailer.SendText(item.TargetOwnerEmail, subject, body); err != nil {
			log.Printf("send project proposal created mail failed: proposal_id=%s err=%v", item.ID, err)
		}
	}()
}

func (rt *Router) notifyProjectProposalReviewed(item domain.ProjectProposal) {
	if rt.mailer == nil || strings.TrimSpace(item.AuthorEmail) == "" {
		return
	}
	subject := "PlayPage 改进提案处理结果"
	statusText := proposalStatusText(item.Status)
	body := fmt.Sprintf("你好，\n\n你提交给作品《%s》的改进提案已有处理结果。\n\n提案标题：%s\n当前状态：%s\n",
		item.TargetProjectName, item.Title, statusText)
	if strings.TrimSpace(item.ReviewNote) != "" {
		body += "作者备注：" + strings.TrimSpace(item.ReviewNote) + "\n"
	}
	if item.Status == "accepted" {
		body += "\n你的改进已经合并到原作品的新版本中。\n"
	}
	body += "\nPlayPage"
	go func() {
		if err := rt.mailer.SendText(item.AuthorEmail, subject, body); err != nil {
			log.Printf("send project proposal reviewed mail failed: proposal_id=%s err=%v", item.ID, err)
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
