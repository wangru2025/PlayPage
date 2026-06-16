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
		Name:                "\u514d\u8d39\u7248",
		MonthlyPriceFen:     0,
		MaxProjects:         5,
		MaxInteractionBytes: 10 * 1024 * 1024,
		MaxMonthlyQueries:   20000,
		MaxMonthlyWrites:    3000,
	},
	PlanLight: {
		Code:                PlanLight,
		Name:                "\u8f7b\u4eab\u7248",
		MonthlyPriceFen:     300,
		MaxProjects:         10,
		MaxInteractionBytes: 30 * 1024 * 1024,
		MaxMonthlyQueries:   100000,
		MaxMonthlyWrites:    10000,
	},
	PlanSupport: {
		Code:                PlanSupport,
		Name:                "\u652f\u6301\u7248",
		MonthlyPriceFen:     600,
		MaxProjects:         30,
		MaxInteractionBytes: 100 * 1024 * 1024,
		MaxMonthlyQueries:   500000,
		MaxMonthlyWrites:    50000,
	},
	PlanAdmin: {
		Code:                PlanAdmin,
		Name:                "\u6700\u9ad8\u7ba1\u7406\u5458",
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
