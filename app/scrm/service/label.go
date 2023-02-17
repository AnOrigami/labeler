package service

import (
	"context"
	"errors"
	"go-admin/app/scrm"
	"go-admin/app/scrm/model"
)

const (
	LabelTypeModel  = "model"
	LabelTypeCall   = "call"
	LabelTypeSeat   = "seat"
	LabelTypeHangup = "hangup"
)

type GetLabelsReq struct {
}

type GetLabelsResp struct {
	ModelLabels  []model.Label `json:"modelLabels"`
	CallLabels   []model.Label `json:"callLabels"`
	SeatLabels   []model.Label `json:"seatLabels"`
	HangupLabels []model.Label `json:"hangupLabels"`
}

func GetLabels(ctx context.Context, req GetLabelsReq) (GetLabelsResp, error) {
	var (
		labels []model.Label
		resp   GetLabelsResp
	)
	db := scrm.GormDB.WithContext(ctx).
		Order("`order`").
		Find(&labels)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error(err.Error())
		return GetLabelsResp{}, errors.New("查询数据库失败")
	}
	for _, v := range labels {
		switch v.Type {
		case LabelTypeModel:
			if resp.ModelLabels == nil {
				resp.ModelLabels = make([]model.Label, 0)
			}
			resp.ModelLabels = append(resp.ModelLabels, v)
		case LabelTypeCall:
			if resp.CallLabels == nil {
				resp.CallLabels = make([]model.Label, 0)
			}
			resp.CallLabels = append(resp.CallLabels, v)
		case LabelTypeSeat:
			if resp.SeatLabels == nil {
				resp.SeatLabels = make([]model.Label, 0)
			}
			resp.SeatLabels = append(resp.SeatLabels, v)
		case LabelTypeHangup:
			if resp.HangupLabels == nil {
				resp.HangupLabels = make([]model.Label, 0)
			}
			resp.HangupLabels = append(resp.HangupLabels, v)
		}
	}
	return resp, nil
}
