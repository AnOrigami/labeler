package service

import (
	"context"
	"go-admin/app/scrm"
	"go-admin/common/log"
	"sync"
	"time"
)

type SeatStatisticService interface {
	Get(seatID int) SeatStatisticInfo
	Run()
	Update(ctx context.Context)
}

type SeatStatisticInfo struct {
	TotalCallAmount   int64
	TotalCallDuration int64
}

type memSeatStatisticDBReceiver struct {
	LockCount    int64
	CallDuration int64
	SeatID       int
}

var SeatStatSvc SeatStatisticService

type MemSeatStatisticService struct {
	SeatInfoMap *map[int]SeatStatisticInfo
	Lock        sync.Mutex
	LoopTime    time.Duration
}

func (svc *MemSeatStatisticService) Get(seatID int) SeatStatisticInfo {
	svc.Lock.Lock()
	defer svc.Lock.Unlock()
	data := (*svc.SeatInfoMap)[seatID]
	return data
}

func (svc *MemSeatStatisticService) Run() {
	for {
		_ = log.WithTracer(context.Background(), PackageName, "seat statistics service", func(ctx context.Context) error {
			svc.Update(ctx)
			return nil
		})
		time.Sleep(svc.LoopTime)
	}
}

func (svc *MemSeatStatisticService) Update(ctx context.Context) {
	var r []memSeatStatisticDBReceiver
	err := scrm.GormDB.WithContext(ctx).
		Table("scrm_call").
		Select("count(*) LockCount,sum(seat_call_duration) CallDuration,seat_id SeatID").
		Where("switch_seat_time is not null").
		Where("date(custom_answer_time)=curdate()").
		Group("seat_id").
		Scan(&r).Error
	if err != nil {
		scrm.Logger().WithContext(ctx).Error("database error when searching seat statistic")
		return
	}
	tempMap := make(map[int]SeatStatisticInfo)
	for _, v := range r {
		tempMap[v.SeatID] = SeatStatisticInfo{
			TotalCallAmount:   v.LockCount,
			TotalCallDuration: v.CallDuration,
		}
	}
	svc.Lock.Lock()
	svc.SeatInfoMap = &tempMap
	svc.Lock.Unlock()
	scrm.Logger().WithContext(ctx).Debugf("seat statistics info: %v", tempMap)
}
