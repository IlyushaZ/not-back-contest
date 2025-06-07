package service

import (
	"context"

	"github.com/IlyushaZ/not-back-contest/pkg/database"
	"github.com/IlyushaZ/not-back-contest/pkg/model"
)

type Sale interface {
	ListPage(ctx context.Context, pageNum, pageSize int) ([]model.Sale, int, error)
}

type SaleGeneric struct {
	SaleRepository database.SaleRepository
}

func (sg *SaleGeneric) ListPage(ctx context.Context, pageNum, pageSize int) ([]model.Sale, int, error) {
	return sg.SaleRepository.GetPage(ctx, pageNum, pageSize)
}
