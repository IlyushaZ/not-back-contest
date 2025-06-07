package handler

type ListPageResp[T any] struct {
	Page  []T `json:"page"`
	Total int `json:"total"`
}
