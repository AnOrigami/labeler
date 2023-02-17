package service

import (
	"context"
	"errors"
	"github.com/go-admin-team/go-admin-core/sdk/pkg"
	"github.com/go-resty/resty/v2"
	"go-admin/app/scrm"
	"go-admin/app/scrm/model"
	"go-admin/common/actions"
	"go-admin/common/log"
	"go-admin/config"
	"sync"
	"time"
)

var quanLiangSession *QuanLiangSession
var restyClient *resty.Client

func QuanLiangSessionInit(ctx context.Context) {
	quanLiangSession = &QuanLiangSession{}
	restyClient = resty.New()
	go func() {
		for {
			time.Sleep(quanLiangSession.LoadToken(ctx))
		}
	}()
}

type QuanLiangSession struct {
	token     string
	tokenLock sync.RWMutex
}

func (r *QuanLiangSession) LoadToken(ctx context.Context) time.Duration {
	next := 10 * time.Second
	_ = log.WithTracer(ctx, PackageName, "WeCom load token", func(ctx context.Context) error {
		var got struct {
			Data struct {
				Data struct {
					AccessToken string `json:"access_token"`
					ExpiresIn   int    `json:"expires_in"`
				}
			}
			QuanLiangCommonResp
		}
		_, err := restyClient.R().SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetBody(map[string]string{
				"app_key":    config.ExtConfig.WeComInteractive.AppKey,
				"app_secret": config.ExtConfig.WeComInteractive.AppSecret,
			}).
			SetResult(&got).
			Post("https://api.aquanliang.com/gateway/qopen/GetAccessToken")
		if err != nil {
			scrm.Logger().WithContext(ctx).Error(err.Error())
			return err
		}
		if got.ErrCode != 0 {
			scrm.Logger().WithContext(ctx).Errorf("圈量error, errcode:%d, errmsg: %s", got.ErrCode, got.ErrMsg)
			return err
		}
		r.tokenLock.Lock()
		defer r.tokenLock.Unlock()
		r.token = got.Data.Data.AccessToken
		next = time.Duration(got.Data.Data.ExpiresIn/2) * time.Second
		return nil
	})
	return next
}

func (r *QuanLiangSession) Get() string {
	r.tokenLock.RLock()
	defer r.tokenLock.RUnlock()
	return r.token
}

type BindSeatAndWeComReq struct {
	WeComName string
}

func BindSeatAndWeCom(ctx context.Context, req BindSeatAndWeComReq) error {
	err := log.WithTracer(ctx, PackageName, "WeCom Bind", func(ctx context.Context) error {
		seatID := actions.GetPermissionFromContext(ctx).UserId
		if req.WeComName == "" {
			err := scrm.GormDB.
				WithContext(ctx).
				Model(&model.Seat{}).
				Where("id = ?", seatID).
				Updates(map[string]any{"we_com": "", "we_com_robot": ""}).
				Error
			if err != nil {
				scrm.Logger().WithContext(ctx).Errorf("gormdb, %s", err.Error())
			}
			return err
		}
		var got struct {
			Data struct {
				HasMore   bool `json:"has_more"`
				RobotList []struct {
					RobotID string `json:"robot_id"`
					UserID  string `json:"user_id"`
					// ... something omitted
				} `json:"robot_list"`
			}
			QuanLiangCommonResp
		}
		offsetPage := 0
		robotID := ""
	EXIT:
		for {
			_, err := restyClient.R().SetContext(ctx).
				SetHeader("Token", quanLiangSession.Get()).
				SetBody(map[string]int{
					"offset": offsetPage * 100,
					"limit":  100,
				}).
				SetResult(&got).
				Post("https://api.aquanliang.com/gateway/qopen/GetPlatformRobotList")
			if err != nil {
				return err
			}
			if got.ErrCode != 0 {
				scrm.Logger().WithContext(ctx).Errorf("圈量error, errcode:%d, errmsg: %s", got.ErrCode, got.ErrMsg)
				return errors.New("圈量平台反馈error")
			}
			for _, v := range got.Data.RobotList {
				if v.UserID == req.WeComName {
					robotID = v.RobotID
					break EXIT
				}
			}
			if !got.Data.HasMore || len(got.Data.RobotList) == 0 {
				break
			}
			offsetPage++
		}
		if robotID == "" {
			return errors.New("尚未在圈量平台注册此企业微信号,请检查")
		}
		err := scrm.GormDB.
			WithContext(ctx).
			Model(&model.Seat{}).
			Where("id = ?", seatID).
			Updates(map[string]any{"we_com": req.WeComName, "we_com_robot": robotID}).
			Error
		if err != nil {
			scrm.Logger().WithContext(ctx).Errorf("gormdb, %s", err.Error())
		}
		return err
	})
	return err
}

type AddFriendReq struct {
	Phone string
}

type QuanLiangCommonResp struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	Hint    string
}

func AddFriend(ctx context.Context, req AddFriendReq) error {
	err := log.WithTracer(ctx, PackageName, "WeCom AddFriend", func(ctx context.Context) error {
		seatID := actions.GetPermissionFromContext(ctx).UserId
		var seat model.Seat
		err := scrm.GormDB.WithContext(ctx).Where("id=?", seatID).First(&seat).Error
		if err != nil {
			scrm.Logger().WithContext(ctx).Errorf("gormdb, %s", err.Error())
			return err
		}
		if seat.WeComRobot == "" {
			return errors.New("尚未绑定")
		}
		var got struct {
			Data struct {
				SerialNo string `json:"serial_no"`
			}
			QuanLiangCommonResp
		}
		_, err = restyClient.R().SetContext(ctx).
			SetHeader("Token", quanLiangSession.Get()).
			SetBody(map[string]any{
				"robot_id": seat.WeComRobot,
				"mark_id":  pkg.GenerateRandomKey20(),
				"ext_user": map[string]string{
					"mobile":             req.Phone,
					"validation_message": config.ExtConfig.WeComInteractive.ValidationMessage,
				},
			}).
			SetResult(&got).
			Post("https://credit.itwq.cn/api/machine/AddExtUserByPhone")
		if err != nil {
			scrm.Logger().WithContext(ctx).Error("绑定失败,可能是网络故障")
			return err
		}
		if got.ErrCode != 0 {
			scrm.Logger().WithContext(ctx).Errorf("圈量error, errcode:%d, errmsg: %s", got.ErrCode, got.ErrMsg)
			return errors.New("圈量平台反馈error")
		}
		return nil
	})
	return err
}

type SearchBindStatusResp struct {
	WeComName string
}

func SearchBindStatus(ctx context.Context) (SearchBindStatusResp, error) {
	var resp SearchBindStatusResp
	return resp, log.WithTracer(ctx, PackageName, "WeCom Search", func(ctx context.Context) error {
		seatID := actions.GetPermissionFromContext(ctx).UserId
		var seat model.Seat
		err := scrm.GormDB.WithContext(ctx).Where("id=?", seatID).First(&seat).Error
		if err != nil {
			scrm.Logger().WithContext(ctx).Errorf("gormdb, %s", err.Error())
			return err
		} else {
			resp.WeComName = seat.WeCom
			return nil
		}
	})
}
