package utils

import (
	"reflect"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PaginationResponse struct {
	TotalPages   int64 `json:"totalPages"`
	TotalItems   int64 `json:"totalItems"`
	CurrentPage  int   `json:"currentPage"`
	ItemsPerPage int   `json:"itemsPerPage"`
}

type SortedPaginationRequest struct {
	Pagination struct {
		Page  int `form:"pagination[page]"`
		Limit int `form:"pagination[limit]"`
	} `form:"pagination"`
	Sort struct {
		Column    string `form:"sort[column]"`
		Direction string `form:"sort[direction]"`
	} `form:"sort"`
	Filters map[string][]string `form:"filters"`
}

func PaginateAndSort(sortedPaginationRequest SortedPaginationRequest, query *gorm.DB, result interface{}, filtersOpt ...map[string][]string) (PaginationResponse, error) {
	var filters map[string][]string
	if len(filtersOpt) > 0 && filtersOpt[0] != nil {
		filters = filtersOpt[0]
	} else {
		filters = sortedPaginationRequest.Filters
	}

	pagination := sortedPaginationRequest.Pagination
	sort := sortedPaginationRequest.Sort

	capitalizedSortColumn := CapitalizeFirstLetter(sort.Column)
	sortField, sortFieldFound := reflect.TypeOf(result).Elem().Elem().FieldByName(capitalizedSortColumn)
	isSortable, _ := strconv.ParseBool(sortField.Tag.Get("sortable"))

	sort.Direction = NormalizeSortDirection(sort.Direction)
	if sortFieldFound && isSortable {
		columnName := CamelCaseToSnakeCase(sort.Column)
		query = query.Clauses(clause.OrderBy{
			Columns: []clause.OrderByColumn{
				{Column: clause.Column{Name: columnName}, Desc: sort.Direction == "desc"},
			},
		})
	}

	// Apply backend facet filters BEFORE paginating (delegated to helper to reduce complexity)
	if len(filters) > 0 && result != nil {
		query = applyFilters(filters, query, result)
	}

	return Paginate(pagination.Page, pagination.Limit, query, result)
}

func Paginate(page int, pageSize int, query *gorm.DB, result interface{}) (PaginationResponse, error) {
	if page < 1 {
		page = 1
	}

	if pageSize < 1 {
		pageSize = 20
	} else if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	var totalItems int64
	if err := query.Count(&totalItems).Error; err != nil {
		return PaginationResponse{}, err
	}

	if err := query.Offset(offset).Limit(pageSize).Find(result).Error; err != nil {
		return PaginationResponse{}, err
	}

	totalPages := (totalItems + int64(pageSize) - 1) / int64(pageSize)
	if totalItems == 0 {
		totalPages = 1
	}

	return PaginationResponse{
		TotalPages:   totalPages,
		TotalItems:   totalItems,
		CurrentPage:  page,
		ItemsPerPage: pageSize,
	}, nil
}

func NormalizeSortDirection(direction string) string {
	d := strings.ToLower(strings.TrimSpace(direction))
	if d != "asc" && d != "desc" {
		return "asc"
	}
	return d
}

func IsValidSortDirection(direction string) bool {
	d := strings.ToLower(strings.TrimSpace(direction))
	return d == "asc" || d == "desc"
}

// applyFilters applies the provided filter map to the GORM query.
// It mirrors the sortable allowlist logic using the model's `filterable:"true"` tag.
// Only supports bool, signed ints, unsigned ints, and string-like fallthrough.
//
//nolint:gocognit
func applyFilters(filters map[string][]string, query *gorm.DB, result interface{}) *gorm.DB {
	if len(filters) == 0 || result == nil {
		return query
	}

	// Derive model type from *[]T or *[]*T
	t := reflect.TypeOf(result)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	modelType := t

	for col, vals := range filters {
		field, ok := modelType.FieldByName(CapitalizeFirstLetter(col))
		if !ok {
			continue
		}
		isFilterable, _ := strconv.ParseBool(field.Tag.Get("filterable"))
		if !isFilterable {
			continue
		}
		columnName := CamelCaseToSnakeCase(col)

		// Unwrap pointer fields
		ft := field.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		kind := ft.Kind()

		// Bool handling
		if kind == reflect.Bool {
			var arr []bool
			for _, s := range vals {
				if b, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(s))); err == nil {
					arr = append(arr, b)
				}
			}
			if len(arr) > 0 {
				query = query.Where(columnName+" IN ?", arr)
			}
			continue
		}

		// Signed integers
		if isSignedIntKind(kind) {
			var arr []int64
			for _, s := range vals {
				if n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
					arr = append(arr, n)
				}
			}
			if len(arr) > 0 {
				query = query.Where(columnName+" IN ?", arr)
			}
			continue
		}

		// Unsigned integers
		if isUnsignedIntKind(kind) {
			var arr []uint64
			for _, s := range vals {
				if n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64); err == nil {
					arr = append(arr, n)
				}
			}
			if len(arr) > 0 {
				query = query.Where(columnName+" IN ?", arr)
			}
			continue
		}

		// Fallback: treat as string values
		var arr []string
		for _, s := range vals {
			if v := strings.TrimSpace(s); v != "" {
				arr = append(arr, v)
			}
		}
		if len(arr) > 0 {
			valsIface := make([]interface{}, len(arr))
			for i, v := range arr {
				valsIface[i] = v
			}
			query = query.Where(clause.IN{Column: clause.Column{Name: columnName}, Values: valsIface})
		}
	}

	return query
}

func isSignedIntKind(k reflect.Kind) bool {
	return k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64
}

func isUnsignedIntKind(k reflect.Kind) bool {
	return k == reflect.Uint || k == reflect.Uint8 || k == reflect.Uint16 || k == reflect.Uint32 || k == reflect.Uint64 || k == reflect.Uintptr
}
