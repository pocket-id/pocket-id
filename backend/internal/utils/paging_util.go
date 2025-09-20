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
}

func PaginateAndSort(sortedPaginationRequest SortedPaginationRequest, query *gorm.DB, result interface{}) (PaginationResponse, error) {
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
