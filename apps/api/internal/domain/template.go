package domain

import "time"

type TemplateConfigField struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Default     string   `json:"default"`
	Placeholder string   `json:"placeholder,omitempty"`
	Help        string   `json:"help,omitempty"`
	Options     []string `json:"options,omitempty"`
}

type TemplateCollectionDefinition struct {
	Name        string        `json:"name"`
	Permissions PermissionSet `json:"permissions"`
	Fields      []FieldSchema `json:"fields"`
}

type ProjectTemplate struct {
	ID                   string                         `json:"id"`
	Slug                 string                         `json:"slug"`
	Name                 string                         `json:"name"`
	Category             string                         `json:"category"`
	CategoryLabel        string                         `json:"categoryLabel"`
	Description          string                         `json:"description"`
	Summary              string                         `json:"summary"`
	Tags                 []string                       `json:"tags"`
	AuthorUserID         string                         `json:"authorUserId,omitempty"`
	AuthorName           string                         `json:"authorName"`
	Source               string                         `json:"source"`
	InteractiveRequired  bool                           `json:"interactiveRequired"`
	AnalyticsRecommended bool                           `json:"analyticsRecommended"`
	UsageCount           int                            `json:"usageCount"`
	ConfigFields         []TemplateConfigField          `json:"configFields"`
	Collections          []TemplateCollectionDefinition `json:"collections"`
	CreatedAtText        string                         `json:"createdAtText"`
	HTMLSource           string                         `json:"-"`
}

type TemplateCreateReleaseInput struct {
	TemplateID string            `json:"templateId"`
	Params     map[string]string `json:"params"`
	ChangeNote string            `json:"changeNote"`
}

type TemplateSubmission struct {
	ID                   string                         `json:"id"`
	AuthorUserID         string                         `json:"authorUserId"`
	AuthorEmail          string                         `json:"authorEmail,omitempty"`
	AuthorName           string                         `json:"authorName"`
	Slug                 string                         `json:"slug"`
	Name                 string                         `json:"name"`
	Category             string                         `json:"category"`
	CategoryLabel        string                         `json:"categoryLabel"`
	Summary              string                         `json:"summary"`
	Description          string                         `json:"description"`
	Tags                 []string                       `json:"tags"`
	InteractiveRequired  bool                           `json:"interactiveRequired"`
	AnalyticsRecommended bool                           `json:"analyticsRecommended"`
	ConfigFields         []TemplateConfigField          `json:"configFields"`
	Collections          []TemplateCollectionDefinition `json:"collections"`
	HTMLSource           string                         `json:"htmlSource,omitempty"`
	SourceType           string                         `json:"sourceType"`
	Status               string                         `json:"status"`
	AdminNote            string                         `json:"adminNote"`
	ReviewedBy           string                         `json:"reviewedBy"`
	ReviewedAt           time.Time                      `json:"reviewedAt"`
	CreatedAt            time.Time                      `json:"createdAt"`
	UpdatedAt            time.Time                      `json:"updatedAt"`
}

type TemplateSubmissionCreateInput struct {
	Slug                 string                         `json:"slug"`
	Name                 string                         `json:"name"`
	Category             string                         `json:"category"`
	CategoryLabel        string                         `json:"categoryLabel"`
	Summary              string                         `json:"summary"`
	Description          string                         `json:"description"`
	Tags                 []string                       `json:"tags"`
	InteractiveRequired  bool                           `json:"interactiveRequired"`
	AnalyticsRecommended bool                           `json:"analyticsRecommended"`
	ConfigFields         []TemplateConfigField          `json:"configFields"`
	Collections          []TemplateCollectionDefinition `json:"collections"`
	HTMLSource           string                         `json:"htmlSource"`
	SourceType           string                         `json:"sourceType"`
}

type TemplateSubmissionReviewInput struct {
	Status    string `json:"status"`
	AdminNote string `json:"adminNote"`
}
