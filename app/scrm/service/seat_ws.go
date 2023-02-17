package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"go-admin/app/scrm"
	"go-admin/common/log"
	"sync"
	"time"
)

type SeatWSEventDataStateChanged struct {
	CheckIn        bool   `json:"checkin"`
	PreReady       bool   `json:"preready"`
	Ready          bool   `json:"ready"`
	Locked         bool   `json:"locked"`
	Projects       []int  `json:"projects"`
	CallID         string `json:"callId"`
	ReadyTimestamp int64  `json:"readyTimestamp"`
}

type SeatStateStore interface {
	Get(ctx context.Context, id string) (SeatWSEventDataStateChanged, error)
	Update(ctx context.Context, id string, f func(data *SeatWSEventDataStateChanged) *SeatWSEventDataStateChanged) (SeatWSEventDataStateChanged, error)
	LockSeat(ctx context.Context, seatID, callID string) (bool, error)
	UnlockSeat(ctx context.Context, seatID, callID string) (bool, error)
}

type redisSeatStateStore struct {
	lock sync.Mutex
}

func (s *redisSeatStateStore) get(ctx context.Context, id string) (SeatWSEventDataStateChanged, error) {
	var data SeatWSEventDataStateChanged
	res, err := scrm.RedisClient.Get(ctx, RedisSeatKey(id)).Result()
	if err != nil {
		if err == redis.Nil {
			return SeatWSEventDataStateChanged{}, nil
		}
		scrm.Logger().WithContext(ctx).Error(err.Error())
		return SeatWSEventDataStateChanged{}, err
	}
	err = json.Unmarshal([]byte(res), &data)
	if err != nil {
		scrm.Logger().WithContext(ctx).Error(err.Error())
		return SeatWSEventDataStateChanged{}, err
	}
	return data, nil
}

func (s *redisSeatStateStore) Get(ctx context.Context, id string) (SeatWSEventDataStateChanged, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.get(ctx, id)
}

func (s *redisSeatStateStore) Update(ctx context.Context, id string, f func(data *SeatWSEventDataStateChanged) *SeatWSEventDataStateChanged) (SeatWSEventDataStateChanged, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	data, err := s.get(ctx, id)
	if err != nil {
		return SeatWSEventDataStateChanged{}, err
	}
	newData := f(&data)
	if newData == nil {
		return data, nil
	}
	if newData.Ready {
		newData.ReadyTimestamp = time.Now().UnixMilli()
	}
	res, err := json.Marshal(newData)
	if err != nil {
		scrm.Logger().WithContext(ctx).Error(err.Error())
		return SeatWSEventDataStateChanged{}, err
	}
	if err := scrm.RedisClient.Set(ctx, RedisSeatKey(id), res, 0).Err(); err != nil {
		scrm.Logger().WithContext(ctx).Error(err.Error())
		return SeatWSEventDataStateChanged{}, err
	}
	return *newData, nil
}

func (s *redisSeatStateStore) LockSeat(ctx context.Context, seatID, callID string) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	data, err := s.get(ctx, seatID)
	if err != nil {
		scrm.Logger().WithContext(ctx).Error(err.Error())
		return false, err
	}
	if !data.Ready || !data.CheckIn || data.Locked {
		return false, nil
	}
	if err := scrm.RedisClient.Set(ctx, RedisCallSeatKey(callID), seatID, 24*time.Hour).Err(); err != nil {
		scrm.Logger().WithContext(ctx).Error(err.Error())
		return false, err
	}
	data.Locked = true
	data.CallID = callID
	data.PreReady = false
	res, err := json.Marshal(data)
	if err != nil {
		scrm.Logger().WithContext(ctx).Error(err.Error())
		return false, err
	}
	if err := scrm.RedisClient.Set(ctx, RedisSeatKey(seatID), res, 0).Err(); err != nil {
		scrm.Logger().WithContext(ctx).Error(err.Error())
		return false, err
	}
	return true, nil
}

func (s *redisSeatStateStore) UnlockSeat(ctx context.Context, seatID, callID string) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if callID != "" {
		scrm.Logger().WithContext(ctx).Debug("UnlockSeat for callID: ", callID)
		v, err := scrm.RedisClient.Get(ctx, RedisCallSeatKey(callID)).Result()
		if err != nil {
			if !errors.Is(err, redis.Nil) {
				scrm.Logger().WithContext(ctx).Error(err.Error())
				return false, err
			}
		}
		if seatID == "" {
			seatID = v
		}
	}
	if seatID == "" {
		return true, nil
	}
	data, err := s.get(ctx, seatID)
	if err != nil {
		scrm.Logger().WithContext(ctx).Error(err.Error())
		return false, err
	}
	if data.Locked && data.CallID == callID {
		data.Locked = false
		data.CallID = ""
		data.ReadyTimestamp = time.Now().UnixMilli()
		res, err := json.Marshal(data)
		if err != nil {
			scrm.Logger().WithContext(ctx).Error(err.Error())
			return false, err
		}
		if err := scrm.RedisClient.Set(ctx, RedisSeatKey(seatID), res, 0).Err(); err != nil {
			scrm.Logger().WithContext(ctx).Error(err.Error())
			return false, err
		}
	}
	if callID != "" {
		err := scrm.RedisClient.Del(ctx, RedisCallSeatKey(callID)).Err()
		if err != nil {
			scrm.Logger().WithContext(ctx).Error(err.Error())
			return false, err
		}
	}
	return true, nil
}

type SeatWSClient struct {
	hub        *SeatHub
	conn       *websocket.Conn
	sendBuffer chan []byte
	UserID     string
	ConnID     string
}

func (c *SeatWSClient) ReceiveLoop(ctx context.Context) {
	ctx = log.NewSpanContext(ctx, PackageName, "seat ws receive loop")
	seatWSUpDownCounter.Add(ctx, 1)
	defer func() {
		c.hub.DeleteClient(c)
		seatWSUpDownCounter.Add(ctx, -1)
		if len(c.hub.userConnClients[c.UserID]) > 0 {
			go c.CheckConnOfUser(ctx)
		}
		_ = c.conn.Close()
	}()
	_ = log.WithTracer(ctx, PackageName, "seat ws receive loop", func(ctx context.Context) error {
		{
			data, err := c.hub.seatStateStore.Get(ctx, c.UserID)
			if err != nil {
				scrm.Logger().WithContext(ctx).Error(err.Error())
				c.hub.SendMessage(ctx, c.UserID, c.ConnID, NewErrorMessage("获取坐席状态失败"))
				return err
			}
			c.hub.SendMessage(ctx, c.UserID, c.ConnID, NewMessage(SeatWSEventStateChanged, data))
		}
		for {
			var exit bool
			_ = log.WithTracer(ctx, PackageName, "seat ws receive loop n", func(ctx context.Context) error {
				var msg MessageOut[json.RawMessage]
				err := c.conn.ReadJSON(&msg)
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						scrm.Logger().WithContext(ctx).Error(err.Error())
					} else {
						scrm.Logger().WithContext(ctx).Info(err.Error())
					}
					// 中断循环
					exit = true
					return nil
				}
				{
					b, _ := json.Marshal(msg)
					scrm.Logger().WithContext(ctx).Debugf("seat:%s ws receive %s", c.UserID, b)
				}
				handlerFactory := SeatWSEventHandlerMap[msg.Event]
				if handlerFactory() == nil {
					err := fmt.Errorf("无法处理事件: %s", msg.Event)
					scrm.Logger().WithContext(ctx).Error(err.Error())
					c.hub.SendMessage(ctx, c.UserID, c.ConnID, NewErrorMessage(err.Error()))
					// 只通知, 不中断循环
					return nil
				}
				handler := handlerFactory()
				if err := json.Unmarshal(msg.Data, handler); err != nil {
					scrm.Logger().WithContext(ctx).Error(err.Error())
					c.hub.SendMessage(ctx, c.UserID, c.ConnID, NewErrorMessage(err.Error()))
					// 只通知, 不中断循环
					return nil
				}
				if err := handler.HandleEvent(ctx, c); err != nil {
					scrm.Logger().WithContext(ctx).Error(err.Error())
					c.hub.SendMessage(ctx, c.UserID, c.ConnID, NewErrorMessage(err.Error()))
					// 只通知, 不中断循环
					return nil
				}
				return nil
			})
			if exit {
				break
			}
		}
		return nil
	})
}

func (c *SeatWSClient) CheckConnOfUser(ctx context.Context) {
	log.WithTracer(ctx, PackageName, "check conn of user", func(ctx context.Context) error {
		time.Sleep(time.Minute)
		if len(c.hub.userConnClients[c.UserID]) == 0 {
			data, err := c.hub.seatStateStore.Update(ctx, c.UserID, func(data *SeatWSEventDataStateChanged) *SeatWSEventDataStateChanged {
				data.CheckIn = false
				data.Ready = false
				seatCheckInUpDownCounter.Add(ctx, -1)
				seatReadinessUpDownCounter.Add(ctx, -1)
				return data
			})
			if err != nil {
				scrm.Logger().WithContext(ctx).Error("store update: ", err.Error())
				return err
			}
			msg := NewMessage(SeatWSEventStateChanged, data)
			c.hub.SendMessage(ctx, c.UserID, "", msg)
			return nil
		}
		return nil
	})
}

func (c *SeatWSClient) SendLoop(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	_ = log.WithTracer(ctx, PackageName, "seat ws send loop", func(ctx context.Context) error {
		for {
			var exit bool
			_ = log.WithTracer(ctx, PackageName, "seat ws send loop n", func(ctx context.Context) error {
				select {
				case msg, ok := <-c.sendBuffer:
					if !ok {
						scrm.Logger().WithContext(ctx).Infof("seat:%s ws channel closed", c.UserID)
						_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
						exit = true
						return nil
					}
					_ = c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
					scrm.Logger().Debugf("seat:%s ws send %s", c.UserID, msg)
					if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
						scrm.Logger().WithContext(ctx).Error(err.Error())
						exit = true
						return nil
					}
				case <-ticker.C:
					_ = c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
					if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						scrm.Logger().WithContext(ctx).Error(err.Error())
						exit = true
						return nil
					}
				}
				return nil
			})
			if exit {
				break
			}
		}
		return nil
	})
}

var DefaultSeatHub = &SeatHub{
	seatStateStore:  &redisSeatStateStore{},
	userConnClients: make(map[string]map[string]*SeatWSClient, 1024),
}

type SeatHub struct {
	seatStateStore  SeatStateStore
	userConnClients map[string]map[string]*SeatWSClient
	lock            sync.Mutex
}

func (s *SeatHub) MakeClient(conn *websocket.Conn, userID, connID string) *SeatWSClient {
	s.lock.Lock()
	defer s.lock.Unlock()
	client := &SeatWSClient{
		hub:        s,
		conn:       conn,
		sendBuffer: make(chan []byte, 128),
		UserID:     userID,
		ConnID:     connID,
	}
	connClients := s.userConnClients[userID]
	if connClients == nil {
		connClients = make(map[string]*SeatWSClient)
		s.userConnClients[userID] = connClients
	}
	connClients[connID] = client
	return client
}

func (s *SeatHub) getClients(userID, connID string) []*SeatWSClient {
	userClients, exists := s.userConnClients[userID]
	if !exists {
		return nil
	}
	if connID != "" {
		client, exists := userClients[connID]
		if !exists {
			return nil
		}
		return []*SeatWSClient{client}
	}
	resp := make([]*SeatWSClient, 0, len(userClients))
	for _, client := range userClients {
		resp = append(resp, client)
	}
	return resp
}

// deleteClient 从 SeatHub 删除 client
// caution: 必须持有锁再调用 deleteClient
func (s *SeatHub) deleteClient(client *SeatWSClient) {
	userClients, exists := s.userConnClients[client.UserID]
	if !exists {
		return
	}
	c, exists := userClients[client.ConnID]
	if !exists {
		return
	}
	delete(s.userConnClients[c.UserID], c.ConnID)
	if len(s.userConnClients[c.UserID]) == 0 {
		delete(s.userConnClients, c.UserID)
	}
	close(c.sendBuffer)
}

func (s *SeatHub) DeleteClient(client *SeatWSClient) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.deleteClient(client)
}

func (s *SeatHub) SendMessage(ctx context.Context, userID, connID string, data interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()
	_ = log.WithTracer(ctx, PackageName, "seat ws send message", func(ctx context.Context) error {
		clients := s.getClients(userID, connID)
		if len(clients) == 0 {
			return nil
		}
		var raw []byte

		if d, ok := data.([]byte); ok {
			raw = d
		} else {
			b, err := json.Marshal(data)
			if err != nil {
				scrm.Logger().WithContext(ctx).Error(err.Error())
			}
			raw = b
		}
		for _, client := range clients {
			select {
			case client.sendBuffer <- raw:
			default:
				s.deleteClient(client)
			}
		}
		return nil
	})
}

type SeatWSEventHandler interface {
	HandleEvent(ctx context.Context, client *SeatWSClient) error
}

var SeatWSEventHandlerMap = map[string]func() SeatWSEventHandler{
	SeatWSEventCheckInChanged: func() SeatWSEventHandler {
		return &SeatWSEventDataCheckIn{}
	},
	SeatWSEventReadinessChanged: func() SeatWSEventHandler {
		return &SeatWSEventDataReadinessChanged{}
	},
}

type SeatWSEventDataCheckIn struct {
	CheckIn bool `json:"checkin"`
}

func (d *SeatWSEventDataCheckIn) HandleEvent(ctx context.Context, client *SeatWSClient) error {
	return log.WithTracer(ctx, PackageName, "SeatWSEventDataCheckIn", func(ctx context.Context) error {
		data, err := client.hub.seatStateStore.Update(ctx, client.UserID, func(data *SeatWSEventDataStateChanged) *SeatWSEventDataStateChanged {
			if data == nil {
				data = &SeatWSEventDataStateChanged{}
			}
			data.PreReady = false
			if data.CheckIn == d.CheckIn {
				return nil
			}
			data.CheckIn = d.CheckIn
			if !data.CheckIn {
				data.Ready = false
				seatCheckInUpDownCounter.Add(ctx, -1)
				seatReadinessUpDownCounter.Add(ctx, -1)
			} else {
				seatCheckInUpDownCounter.Add(ctx, 1)
			}
			return data
		})
		if err != nil {
			scrm.Logger().WithContext(ctx).Error(err.Error())
			client.hub.SendMessage(ctx, client.UserID, client.ConnID, NewErrorMessage(err.Error()))
			return err
		}
		msg := NewMessage(SeatWSEventStateChanged, data)
		client.hub.SendMessage(ctx, client.UserID, "", msg)
		return nil
	})
}

type SeatWSEventDataReadinessChanged struct {
	Ready    bool  `json:"ready"`
	Projects []int `json:"projects"`
}

func (d *SeatWSEventDataReadinessChanged) HandleEvent(ctx context.Context, client *SeatWSClient) error {
	return log.WithTracer(ctx, PackageName, "SeatWSEventDataReadinessChanged", func(ctx context.Context) error {
		var outerErr error
		data, err := client.hub.seatStateStore.Update(ctx, client.UserID, func(data *SeatWSEventDataStateChanged) *SeatWSEventDataStateChanged {
			if data == nil {
				data = &SeatWSEventDataStateChanged{}
			}
			data.PreReady = false
			if data.Ready == d.Ready && SliceEqual(data.Projects, d.Projects) {
				return nil
			}
			data.Ready = d.Ready
			data.Projects = d.Projects
			if data.Ready && (!data.CheckIn || len(data.Projects) == 0) {
				outerErr = errors.New("请签入后选择项目示闲")
				return nil
			}
			if data.Ready {
				seatReadinessUpDownCounter.Add(ctx, 1)
			} else {
				seatReadinessUpDownCounter.Add(ctx, -1)
			}
			return data
		})
		if err != nil {
			scrm.Logger().WithContext(ctx).Error(err.Error())
			client.hub.SendMessage(ctx, client.UserID, client.ConnID, NewErrorMessage(err.Error()))
			return err
		}
		if outerErr != nil {
			client.hub.SendMessage(ctx, client.UserID, client.ConnID, NewErrorMessage(outerErr.Error()))
			return err
		}
		msg := NewMessage(SeatWSEventStateChanged, data)
		client.hub.SendMessage(ctx, client.UserID, "", msg)
		return nil
	})
}
