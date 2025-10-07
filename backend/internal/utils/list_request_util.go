package utils

import (
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PaginationResponse struct {
	TotalPages   int64 `json:"totalPages"`
	TotalItems   int64 `json:"totalItems"`
	CurrentPage  int   `json:"currentPage"`
	ItemsPerPage int   `json:"itemsPerPage"`
}

type ListRequestOptions struct {
	Pagination struct {
		Page  int `form:"pagination[page]"`
		Limit int `form:"pagination[limit]"`
	} `form:"pagination"`
	Sort struct {
		Column    string `form:"sort[column]"`
		Direction string `form:"sort[direction]"`
	} `form:"sort"`
	Filters map[string][]any
}

type FieldMeta struct {
	ColumnName   string
	IsSortable   bool
	IsFilterable bool
}

func ParseListRequestOptions(ctx *gin.Context) (listRequestOptions ListRequestOptions) {
	if err := ctx.ShouldBindQuery(&listRequestOptions); err != nil {
		return listRequestOptions
	}

	listRequestOptions.Filters = parseNestedFilters(ctx)
	return listRequestOptions
}

func PaginateFilterAndSort(params ListRequestOptions, query *gorm.DB, result interface{}) (PaginationResponse, error) {
	meta := extractModelMetadata(result)

	query = applyFilters(params.Filters, query, meta)
	query = applySorting(params.Sort.Column, params.Sort.Direction, query, meta)

	return Paginate(params.Pagination.Page, params.Pagination.Limit, query, result)
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

	var totalItems int64
	if err := query.Count(&totalItems).Error; err != nil {
		return PaginationResponse{}, err
	}

	totalPages := (totalItems + int64(pageSize) - 1) / int64(pageSize)
	if totalItems == 0 {
		totalPages = 1
	}

	if int64(page) > totalPages {
		page = int(totalPages)
	}

	offset := (page - 1) * pageSize

	if err := query.Offset(offset).Limit(pageSize).Find(result).Error; err != nil {
		return PaginationResponse{}, err
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

// parseNestedFilters handles ?filters[field][0]=val1&filters[field][1]=val2
func parseNestedFilters(ctx *gin.Context) map[string][]any {
	result := make(map[string][]any)
	query := ctx.Request.URL.Query()

	for key, values := range query {
		if !strings.HasPrefix(key, "filters[") {
			continue
		}

		// Example key: "filters[disabled][0]"
		field := strings.TrimPrefix(key, "filters[")
		field = strings.SplitN(field, "][", 2)[0]

		for _, v := range values {
			result[field] = append(result[field], ConvertStringToType(v))
		}
	}

	return result
}

// applyFilters applies filtering to the GORM query based on the provided filters
func applyFilters(filters map[string][]any, query *gorm.DB, meta map[string]FieldMeta) *gorm.DB {
	for key, values := range filters {
		if key == "" || len(values) == 0 {
			continue
		}

		fieldName := CapitalizeFirstLetter(key)
		fieldMeta, ok := meta[fieldName]
		if !ok || !fieldMeta.IsFilterable {
			continue
		}

		query = query.Where(fieldMeta.ColumnName+" IN ?", values)
	}
	return query
}

// applySorting applies sorting to the GORM query based on the provided column and direction
func applySorting(sortColumn string, sortDirection string, query *gorm.DB, meta map[string]FieldMeta) *gorm.DB {
	fieldName := CapitalizeFirstLetter(sortColumn)
	fieldMeta, ok := meta[fieldName]
	if !ok || !fieldMeta.IsSortable {
		return query
	}

	sortDirection = NormalizeSortDirection(sortDirection)

	query = query.Clauses(clause.OrderBy{
		Columns: []clause.OrderByColumn{
			{Column: clause.Column{Name: fieldMeta.ColumnName}, Desc: sortDirection == "desc"},
		},
	})
	return query
}

// extractModelMetadata extracts FieldMeta from the model struct using reflection
func extractModelMetadata(model interface{}) map[string]FieldMeta {
	meta := make(map[string]FieldMeta)

	modelType := reflect.TypeOf(model).Elem().Elem()

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		name := field.Name

		meta[name] = FieldMeta{
			ColumnName:   CamelCaseToSnakeCase(name),
			IsSortable:   field.Tag.Get("sortable") == "true",
			IsFilterable: field.Tag.Get("filterable") == "true",
		}
	}

	return meta
}
