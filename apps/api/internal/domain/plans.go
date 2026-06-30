package domain

type PlanSpec struct {
	Code                string `json:"code"`
	Name                string `json:"name"`
	MonthlyPriceFen     int    `json:"monthlyPriceFen"`
	MaxProjects         int    `json:"maxProjects"`
	MaxInteractionBytes int64  `json:"maxInteractionBytes"`
	MaxMonthlyQueries   int64  `json:"maxMonthlyQueries"`
	MaxMonthlyWrites    int64  `json:"maxMonthlyWrites"`
}

const (
	RoleUser       = "user"
	RoleAdmin      = "admin"
	RoleSuperAdmin = "super_admin"

	PlanFree    = "free"
	PlanLight   = "light"
	PlanSupport = "support"
	PlanAdmin   = "admin"
)

var planTable = map[string]PlanSpec{
	PlanFree: {
		Code:                PlanFree,
		Name:                "免费版",
		MonthlyPriceFen:     0,
		MaxProjects:         5,
		MaxInteractionBytes: 10 * 1024 * 1024,
		MaxMonthlyQueries:   20000,
		MaxMonthlyWrites:    3000,
	},
	PlanLight: {
		Code:                PlanLight,
		Name:                "轻享版",
		MonthlyPriceFen:     300,
		MaxProjects:         10,
		MaxInteractionBytes: 30 * 1024 * 1024,
		MaxMonthlyQueries:   100000,
		MaxMonthlyWrites:    10000,
	},
	PlanSupport: {
		Code:                PlanSupport,
		Name:                "支持版",
		MonthlyPriceFen:     600,
		MaxProjects:         30,
		MaxInteractionBytes: 100 * 1024 * 1024,
		MaxMonthlyQueries:   500000,
		MaxMonthlyWrites:    50000,
	},
	PlanAdmin: {
		Code:                PlanAdmin,
		Name:                "最高管理员",
		MonthlyPriceFen:     0,
		MaxProjects:         10000000,
		MaxInteractionBytes: 1 << 50,
		MaxMonthlyQueries:   1 << 50,
		MaxMonthlyWrites:    1 << 50,
	},
}

func PlanByCode(code string) PlanSpec {
	if plan, ok := planTable[code]; ok {
		return plan
	}
	return planTable[PlanFree]
}

func PublicPlans() []PlanSpec {
	return []PlanSpec{
		planTable[PlanFree],
		planTable[PlanLight],
		planTable[PlanSupport],
	}
}

func IsAdminRole(role string) bool {
	return role == RoleAdmin || role == RoleSuperAdmin
}

func IsSuperAdminRole(role string) bool {
	return role == RoleSuperAdmin
}

func DailyEmailCodeLimit(planCode string, role string) int {
	if IsAdminRole(role) {
		return 10000
	}
	switch planCode {
	case PlanSupport:
		return 100
	case PlanLight:
		return 50
	default:
		return 20
	}
}
