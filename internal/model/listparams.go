package model

type ListParams struct {
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string // "asc" or "desc"
	Filters   map[string]string
}
