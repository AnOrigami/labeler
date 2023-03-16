package service

import (
	"context"
	"go-admin/app/admin/models"
)

func (svc *LabelerService) GetUserList(ctx context.Context) ([]models.SysUser, int, error) {
	var users []models.SysUser
	db := svc.GormDB.WithContext(ctx).
		Find(&users)
	if err := db.Error; err != nil {
		return users, 0, err
	}
	return users, len(users), nil
}
