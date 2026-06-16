package http

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"ai-static-host/api/internal/domain"
)

const maxRecordListLimit = 200

var recordFieldPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type recordQuery struct {
	Filters []recordFilter
	Sorts   []recordSort
	Limit   int
	Offset  int
}

type recordFilter struct {
	Field string
	Op    string
	Value string
}

type recordSort struct {
	Field string
	Desc  bool
}

func parseRecordQuery(r *http.Request) (recordQuery, error) {
	values := r.URL.Query()
	q := recordQuery{Limit: maxRecordListLimit}

	if raw := strings.TrimSpace(values.Get("limit")); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 {
			return q, fmt.Errorf("limit 必须是正整数")
		}
		if limit > maxRecordListLimit {
			limit = maxRecordListLimit
		}
		q.Limit = limit
	}
	if raw := strings.TrimSpace(values.Get("offset")); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return q, fmt.Errorf("offset 必须是非负整数")
		}
		q.Offset = offset
	}

	for _, raw := range values["where"] {
		filter, err := parseWhereFilter(raw)
		if err != nil {
			return q, err
		}
		q.Filters = append(q.Filters, filter)
	}
	for _, raw := range values["filter"] {
		filter, err := parseCompatFilter(raw)
		if err != nil {
			return q, err
		}
		q.Filters = append(q.Filters, filter)
	}
	for key, list := range values {
		if strings.HasPrefix(key, "data.") && len(list) > 0 {
			field := strings.TrimPrefix(key, "data.")
			if !validRecordField(field) {
				return q, fmt.Errorf("查询字段名不正确")
			}
			q.Filters = append(q.Filters, recordFilter{Field: field, Op: "eq", Value: list[len(list)-1]})
		}
	}

	sortText := strings.TrimSpace(values.Get("sort"))
	if sortText == "" {
		sortText = "-createdAt"
	}
	for _, part := range strings.Split(sortText, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		desc := false
		if strings.HasPrefix(part, "-") {
			desc = true
			part = strings.TrimPrefix(part, "-")
		} else if strings.HasPrefix(part, "+") {
			part = strings.TrimPrefix(part, "+")
		}
		part = strings.TrimPrefix(part, "data.")
		if !validSortableRecordField(part) {
			return q, fmt.Errorf("排序字段不支持")
		}
		q.Sorts = append(q.Sorts, recordSort{Field: part, Desc: desc})
	}
	return q, nil
}

func parseWhereFilter(raw string) (recordFilter, error) {
	parts := strings.SplitN(strings.TrimSpace(raw), ":", 3)
	if len(parts) != 3 {
		return recordFilter{}, fmt.Errorf("where 格式应为 field:op:value")
	}
	field := strings.TrimPrefix(strings.TrimSpace(parts[0]), "data.")
	op := strings.ToLower(strings.TrimSpace(parts[1]))
	value, _ := url.QueryUnescape(strings.TrimSpace(parts[2]))
	if !validRecordField(field) || !validRecordFilterOp(op) {
		return recordFilter{}, fmt.Errorf("where 查询条件不正确")
	}
	return recordFilter{Field: field, Op: op, Value: trimQuotes(value)}, nil
}

func parseCompatFilter(raw string) (recordFilter, error) {
	text := strings.TrimSpace(raw)
	re := regexp.MustCompile(`^data\.([A-Za-z_][A-Za-z0-9_]*)\s*(==|=|!=|>=|<=|>|<|contains)\s*(.+)$`)
	matches := re.FindStringSubmatch(text)
	if len(matches) != 4 {
		return recordFilter{}, fmt.Errorf("filter 只支持 data.field==value 这类简单条件")
	}
	op := map[string]string{"==": "eq", "=": "eq", "!=": "ne", ">": "gt", ">=": "gte", "<": "lt", "<=": "lte", "contains": "contains"}[matches[2]]
	value, _ := url.QueryUnescape(strings.TrimSpace(matches[3]))
	return recordFilter{Field: matches[1], Op: op, Value: trimQuotes(value)}, nil
}

func applyRecordQuery(items []domain.Record, q recordQuery) []domain.Record {
	filtered := make([]domain.Record, 0, len(items))
	for _, item := range items {
		if recordMatches(item, q.Filters) {
			filtered = append(filtered, item)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		for _, s := range q.Sorts {
			cmp := compareRecordField(filtered[i], filtered[j], s.Field)
			if cmp == 0 {
				continue
			}
			if s.Desc {
				return cmp > 0
			}
			return cmp < 0
		}
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})
	if q.Offset >= len(filtered) {
		return []domain.Record{}
	}
	end := q.Offset + q.Limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[q.Offset:end]
}

func recordMatches(item domain.Record, filters []recordFilter) bool {
	for _, f := range filters {
		if !matchRecordValue(item.Data[f.Field], f.Op, f.Value) {
			return false
		}
	}
	return true
}

func matchRecordValue(value any, op, expected string) bool {
	actual := fmt.Sprint(value)
	switch op {
	case "eq":
		return actual == expected
	case "ne":
		return actual != expected
	case "contains":
		return strings.Contains(actual, expected)
	case "gt", "gte", "lt", "lte":
		af, aok := toFloat(actual)
		bf, bok := toFloat(expected)
		if aok && bok {
			switch op {
			case "gt":
				return af > bf
			case "gte":
				return af >= bf
			case "lt":
				return af < bf
			case "lte":
				return af <= bf
			}
		}
		switch op {
		case "gt":
			return actual > expected
		case "gte":
			return actual >= expected
		case "lt":
			return actual < expected
		case "lte":
			return actual <= expected
		}
	}
	return false
}

func compareRecordField(a, b domain.Record, field string) int {
	if field == "createdAt" {
		return compareTime(a.CreatedAt, b.CreatedAt)
	}
	if field == "updatedAt" {
		return compareTime(a.UpdatedAt, b.UpdatedAt)
	}
	if field == "id" {
		return strings.Compare(a.ID, b.ID)
	}
	av := fmt.Sprint(a.Data[field])
	bv := fmt.Sprint(b.Data[field])
	af, aok := toFloat(av)
	bf, bok := toFloat(bv)
	if aok && bok {
		if af < bf {
			return -1
		}
		if af > bf {
			return 1
		}
		return 0
	}
	return strings.Compare(av, bv)
}

func compareTime(a, b time.Time) int {
	if a.Before(b) {
		return -1
	}
	if a.After(b) {
		return 1
	}
	return 0
}

func toFloat(s string) (float64, bool) {
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return n, err == nil
}

func trimQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func validRecordField(field string) bool {
	return recordFieldPattern.MatchString(field)
}

func validSortableRecordField(field string) bool {
	return field == "id" || field == "createdAt" || field == "updatedAt" || validRecordField(field)
}

func validRecordFilterOp(op string) bool {
	switch op {
	case "eq", "ne", "contains", "gt", "gte", "lt", "lte":
		return true
	default:
		return false
	}
}
