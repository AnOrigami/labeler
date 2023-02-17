package model

import (
	"database/sql"
	"go-admin/common/models"
	"go-admin/common/util"
)

type Sentence struct {
	ID     int    `json:"id" gorm:"primaryKey;autoIncrement;comment:主键编码"`
	CallID string `json:"callId"`
	Role   int    `json:"role" gorm:"size:10"`
	Index  int    `json:"index"`
	Text   string `json:"text"`

	models.ModelTime
	models.ControlBy
}

func (Sentence) TableName() string {
	return "scrm_sentence"
}

type Call struct {
	ID                    string         `json:"id" gorm:"primaryKey;size:191;"`
	Phone                 string         `json:"phone"`
	OrderID               int            `json:"orderId"`
	Order                 *Order         `json:"-"`
	LabelID               sql.NullInt64  `json:"-" gorm:"comment:模型标签;"`
	Label                 *Label         `json:"-"`
	CallLabelID           sql.NullInt64  `json:"-"`
	CallLabel             *Label         `json:"-"`
	SeatLabelID           sql.NullInt64  `json:"-"`
	SeatLabel             *Label         `json:"-"`
	HangupLabelID         sql.NullInt64  `json:"-"`
	HangupLabel           *Label         `json:"-"`
	Sentences             []Sentence     `json:"sentences"`
	Comment               string         `json:"comment" gorm:"type:text;"`
	Detail                string         `json:"detail" gorm:"type:text;"`
	AudioFile             string         `json:"audioFile"`
	SeatID                int            `json:"seatId"`
	DialUpCustomTime      sql.NullTime   `gorm:"comment:CTI呼出给客户的时间点;"`
	DialUpSeatTime        sql.NullTime   `gorm:"comment:CTI呼出给坐席的时间点;"`
	HangUpTime            sql.NullTime   `gorm:"comment:CTI挂断电话的时间点;"`
	CustomAnswerTime      sql.NullTime   `gorm:"comment:客户接起电话的时间点;"`
	SeatAnswerTime        sql.NullTime   `gorm:"comment:坐席接起电话的时间点;"`
	SwitchSeatTime        sql.NullTime   `gorm:"comment:机器人决定转接人工的时间点;"`
	Line                  sql.NullString `json:"line" gorm:"size:20;"`
	CustomRingingDuration int64          `gorm:"not null;comment:当未接通时，从呼出到挂断的时长;"`
	SeatRingingDuration   int64          `gorm:"not null;comment:从呼叫坐席到坐席接起的时长"`
	AICallDuration        int64          `gorm:"not null;comment:从客户接起到机器人转接的时长;"`
	SeatCallDuration      int64          `gorm:"not null;comment:从坐席接起到挂断的时长;"`
	SwitchingDuration     int64          `gorm:"not null;comment:从机器人转接到坐席接起的时长（用于控制并发量）;"`
	TotalCallDuration     int64          `gorm:"not null;comment:从客户接起到挂断的时长（用于和纯人工外呼方式比较;"`

	models.ModelTime
	models.ControlBy
}

func (*Call) TableName() string {
	return "scrm_call"
}

func (c *Call) UpdateDuration() {
	switch { // CustomRingingDuration
	case c.CustomAnswerTime.Valid && c.DialUpCustomTime.Valid:
		c.CustomRingingDuration = util.DurationSecs(c.CustomAnswerTime.Time.Sub(c.DialUpCustomTime.Time))
	case c.HangUpTime.Valid && c.DialUpCustomTime.Valid:
		c.CustomRingingDuration = util.DurationSecs(c.HangUpTime.Time.Sub(c.DialUpCustomTime.Time))
	}
	switch { // SeatRingingDuration
	case c.SeatAnswerTime.Valid && c.DialUpSeatTime.Valid:
		c.SeatRingingDuration = util.DurationSecs(c.SeatAnswerTime.Time.Sub(c.DialUpSeatTime.Time))
	case c.HangUpTime.Valid && c.DialUpSeatTime.Valid:
		c.SeatRingingDuration = util.DurationSecs(c.HangUpTime.Time.Sub(c.DialUpSeatTime.Time))
	}
	switch { // AICallDuration
	case c.SeatAnswerTime.Valid && c.CustomAnswerTime.Valid:
		c.AICallDuration = util.DurationSecs(c.SeatAnswerTime.Time.Sub(c.CustomAnswerTime.Time))
	case c.HangUpTime.Valid && c.CustomAnswerTime.Valid:
		c.AICallDuration = util.DurationSecs(c.HangUpTime.Time.Sub(c.CustomAnswerTime.Time))
	}
	switch { // SeatCallDuration
	case c.HangUpTime.Valid && c.SeatAnswerTime.Valid:
		c.SeatCallDuration = util.DurationSecs(c.HangUpTime.Time.Sub(c.SeatAnswerTime.Time))
	}
	switch { // SwitchingDuration
	case c.SeatAnswerTime.Valid && c.SwitchSeatTime.Valid:
		c.SwitchingDuration = util.DurationSecs(c.SeatAnswerTime.Time.Sub(c.SwitchSeatTime.Time))
	case c.HangUpTime.Valid && c.SwitchSeatTime.Valid:
		c.SwitchingDuration = util.DurationSecs(c.HangUpTime.Time.Sub(c.SwitchSeatTime.Time))
	}
	switch { // TotalCallDuration
	case c.HangUpTime.Valid && c.CustomAnswerTime.Valid:
		c.TotalCallDuration = util.DurationSecs(c.HangUpTime.Time.Sub(c.CustomAnswerTime.Time))
	}
}
