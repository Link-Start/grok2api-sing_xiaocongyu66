package repository

import "time"

type SortDirection string

const (
	SortAscending  SortDirection = "asc"
	SortDescending SortDirection = "desc"
)

// SortQuery 只携带应用层验证后的领域字段名，持久化层负责映射为固定 SQL 表达式。
type SortQuery struct {
	Field     string
	Direction SortDirection
}

func IsValidSort(sort SortQuery, fields ...string) bool {
	if sort.Field == "" {
		return sort.Direction == ""
	}
	if sort.Direction != SortAscending && sort.Direction != SortDescending {
		return false
	}
	for _, field := range fields {
		if sort.Field == field {
			return true
		}
	}
	return false
}

// PageQuery 表示管理端页码列表的稳定查询边界。
type PageQuery struct {
	Offset int
	Limit  int
	Search string
	Sort   SortQuery
}

type AccountListFilter struct {
	Provider    string
	QuotaType   string
	Status      string
	Refreshable *bool
	Now         time.Time
}

type AccountListQuery struct {
	Page   PageQuery
	Filter AccountListFilter
}

type AccountSummary struct {
	Provider       string
	Total          int64
	Available      int64
	Cooldown       int64
	WaitingReset   int64
	Probing        int64
	Disabled       int64
	ReauthRequired int64
}

// WebPoolSummary is the basic / super / heavy account distribution for Grok Web.
type WebPoolSummary struct {
	Basic WebPoolBucket
	Super WebPoolBucket
	Heavy WebPoolBucket
	Auto  WebPoolBucket
}

// WebPoolBucket counts accounts in one Web tier pool.
type WebPoolBucket struct {
	Total     int64
	Available int64
}

// ConsoleQuotaSummary is the local console free-model quota rotation snapshot.
type ConsoleQuotaSummary struct {
	Total      int64
	Available  int64
	Healthy    int64 // remaining above rotate threshold, no timer
	Rotating   int64 // timer started, remaining > 0
	Exhausted  int64 // remaining == 0 (waiting reset)
	Remaining  int64 // sum of remaining requests across enabled active accounts
	Capacity   int64 // sum of total window capacity across enabled active accounts
}

type ModelListFilter struct {
	Provider string
	Enabled  *bool
}

type ModelListQuery struct {
	Page   PageQuery
	Filter ModelListFilter
}

type ClientKeyListFilter struct {
	Status     string
	ModelScope string
	Now        time.Time
}

type ClientKeyListQuery struct {
	Page   PageQuery
	Filter ClientKeyListFilter
}

type AuditListFilter struct {
	Model   string
	Status  string
	Mode    string
	Key     string
	Account string
}

type AuditCursorQuery struct {
	Cursor *SortCursor
	Limit  int
	Search string
	Start  time.Time
	End    time.Time
	Sort   SortQuery
	Filter AuditListFilter
}

// SortCursor 是复合游标的持久化边界。Value 类型由 SortQuery.Field 对应的固定映射决定。
type SortCursor struct {
	ID    uint64
	Value any
}

type AuditSummaryQuery struct {
	Search string
	Start  time.Time
	End    time.Time
	Filter AuditListFilter
}
