package http

import (
	"context"
	"fmt"
	"time"

	"ai-static-host/api/internal/domain"
)

func currentMonthKey(now time.Time) string {
	return now.UTC().Format("2006-01")
}

func (rt *Router) checkProjectCreationAllowed(user domain.User) error {
	if domain.IsSuperAdminRole(user.Role) {
		return nil
	}

	plan := domain.PlanByCode(user.PlanCode)
	projects, err := rt.store.ListProjects(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("读取作品数量失败")
	}
	if len(projects) >= plan.MaxProjects {
		return fmt.Errorf("你当前的套餐最多可以创建 %d 个作品，如需更多作品，请升级套餐", plan.MaxProjects)
	}
	return nil
}

func (rt *Router) checkProjectUsageAllowed(access domain.PublicProjectAccess, queryDelta, writeDelta int64) error {
	if domain.IsSuperAdminRole(access.OwnerRole) {
		return nil
	}

	plan := domain.PlanByCode(access.OwnerPlan)
	monthKey := currentMonthKey(time.Now())
	usage, err := rt.store.GetMonthlyProjectUsage(context.Background(), access.Project.ID, monthKey)
	if err != nil {
		return fmt.Errorf("读取作品用量失败")
	}
	if usage.QueryCount+queryDelta > plan.MaxMonthlyQueries {
		return fmt.Errorf("这个作品本月的互动查询次数已经用完了")
	}
	if usage.WriteCount+writeDelta > plan.MaxMonthlyWrites {
		return fmt.Errorf("这个作品本月的互动写入次数已经用完了")
	}
	return nil
}

func (rt *Router) addProjectUsage(projectID string, queryDelta, writeDelta int64) {
	if queryDelta == 0 && writeDelta == 0 {
		return
	}
	_, _ = rt.store.IncrementProjectUsage(context.Background(), projectID, currentMonthKey(time.Now()), queryDelta, writeDelta)
}
