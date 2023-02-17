package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/tjfoc/gmsm/sm3"
	"github.com/xuri/excelize/v2"
	"go-admin/app/scrm"
	"go-admin/common/util"
	"mime/multipart"
	"strings"
)

var (
	ErrSheetFormat = errors.New("表格式异常")
	ErrFileFormat  = errors.New("文件异常")
)

type CallHistoryRow struct {
	Index                 string
	ID                    string
	OrderID               string
	Phone                 string
	Project               string
	CreatedAt             string
	Sentences             string
	ModelLabel            string
	CallLabel             string
	SeatLabel             string
	HangupLabel           string
	Comment               string
	DialUpCustomTime      string
	CustomAnswerTime      string
	SwitchSeatTime        string
	DialUpSeatTime        string
	SeatAnswerTime        string
	HangUpTime            string
	CustomRingingDuration string
	SeatRingingDuration   string
	AICallDuration        string
	SeatCallDuration      string
	SwitchingDuration     string
	TotalCallDuration     string
	SeatUserName          string
	SeatName              string
	Line                  string
}

type EncryptionPhoneRow struct {
	WangZhanDaiMa                                string
	KeHuXinMing                                  string
	ZhengJianHaoMa                               string
	ShouJiHaoMa                                  string
	LuRuShiJian                                  string
	ChuShenJieGuoDaiMa                           string
	ChuShenJieShuRiQi                            string
	ShenPiJueDingBiaoShi                         string
	ShenQingJueDingWanChengRiQi                  string
	DangYuePiHeDangYueZhuXiaoBiaoZhi             string
	XinYongHuBiaoZhi                             string
	YouXiaoKeHuBiaoZhi                           string
	KaPianShouShuaRiQi                           string
	YouXiaoChuShen180TianBiaoShi                 string
	YouXiaoChuShen60TianXinYongHuShouShuaBiaoShi string
	YouXiaoChuShen60TianXinYongHuShouShuaRiQi    string
	TuiGuangRenYuanDaiMa                         string
}

type ExportRow struct {
	CallHistoryRow
	EncryptionPhoneRow
	Match bool
}

type ExportNotConsumedPhoneReq struct {
	Files []*multipart.FileHeader
}

type ExportNotConsumedPhoneResp struct {
	File     *string `json:"file"`
	Filename string  `json:"filename"`
}

func ExportNotConsumedPhone(ctx context.Context, req ExportNotConsumedPhoneReq) (ExportNotConsumedPhoneResp, error) {
	var resp ExportNotConsumedPhoneResp

	callsMap, err := ReadCallHistoryExcel(req.Files[0])
	if err != nil {
		return resp, err
	}
	phones, err := ReadEncryptionPhoneExcel(req.Files[1])
	if err != nil {
		return resp, err
	}

	exportRows := make([]ExportRow, len(phones))
	for i, p := range phones {
		row := ExportRow{CallHistoryRow{}, p, false}
		if v, ok := callsMap[p.ShouJiHaoMa]; ok {
			row.CallHistoryRow = v
			row.Match = true
		}
		exportRows[i] = row
	}

	columns := []string{"网址代码", "客户姓名", "证件号码", "手机号码", "录入时间", "初审结果代码", "初审结束日期",
		"审批决定标识", "申请决定完成日期", "当月批核当月注销标志", "新户标志", "有效客户标志", "卡片首刷日期", "有效初审180天标识",
		"有效初审60天内新户首刷标识", "有效初审60天内新户首刷的首刷日期", "推广人员代码", "成功匹配手机号",
		"序号", "通话编号", "工单编号", "电话号码", "项目编号", "创建时间", "通话记录", "模型标签", "通话标签", "坐席标签", "挂断标签",
		"备注", "呼出给客户时刻", "客户接起时刻", "转换时刻", "呼出给坐席时刻", "坐席接起时刻", "挂断时刻", "客户响铃时长", "坐席响铃时长",
		"机器人通话时长", "坐席通话时长", "客户等待转接时长", "综合通话时长", "坐席用户名", "坐席名", "线路",
	}
	data, filename, err := util.CreateExcelFile(
		phoneRowToSlice(exportRows),
		columns,
		"首刷",
	)
	if err != nil {
		return resp, err
	}
	resp.File = data
	resp.Filename = filename
	return resp, nil
}

func ReadEncryptionPhoneExcel(file *multipart.FileHeader) ([]EncryptionPhoneRow, error) {
	phoneFile, err := file.Open()
	if err != nil {
		scrm.Logger().Error("open file error", err.Error())
		return nil, ErrFileFormat
	}
	defer func() {
		if err = phoneFile.Close(); err != nil {
			scrm.Logger().Error(err.Error())
		}
	}()
	f, err := excelize.OpenReader(phoneFile)
	if err != nil {
		scrm.Logger().Error(err.Error())
		return nil, err
	}
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		scrm.Logger().Error("no sheet")
		return nil, ErrSheetFormat
	}
	sheet := sheets[0]
	rows, err := f.GetRows(sheet)
	if err != nil {
		scrm.Logger().Error(err.Error())
		return nil, ErrSheetFormat
	}
	cols, err := f.GetCols(sheet)
	if err != nil {
		scrm.Logger().Error(err.Error())
		return nil, ErrSheetFormat
	}
	colLen := len(cols)
	phoneRows := make([]EncryptionPhoneRow, 0)
	for i := range rows {
		if i == 0 {
			continue
		}
		row := make([]string, colLen)
		for j := range row {
			val, err := f.GetCellValue(sheet, fmt.Sprintf("%s%d", util.ColMap[j+1], i+1))
			if err != nil {
				scrm.Logger().Error(err.Error())
				return nil, ErrSheetFormat
			}
			row[j] = val
		}
		if len(row) < 17 {
			scrm.Logger().Errorf("excel row length:%d", len(row))
			return nil, ErrSheetFormat
		}
		phoneRows = append(phoneRows, EncryptionPhoneRow{
			WangZhanDaiMa:                    row[0],
			KeHuXinMing:                      row[1],
			ZhengJianHaoMa:                   row[2],
			ShouJiHaoMa:                      row[3],
			LuRuShiJian:                      row[4],
			ChuShenJieGuoDaiMa:               row[5],
			ChuShenJieShuRiQi:                row[6],
			ShenPiJueDingBiaoShi:             row[7],
			ShenQingJueDingWanChengRiQi:      row[8],
			DangYuePiHeDangYueZhuXiaoBiaoZhi: row[9],
			XinYongHuBiaoZhi:                 row[10],
			YouXiaoKeHuBiaoZhi:               row[11],
			KaPianShouShuaRiQi:               row[12],
			YouXiaoChuShen180TianBiaoShi:     row[13],
			YouXiaoChuShen60TianXinYongHuShouShuaBiaoShi: row[14],
			YouXiaoChuShen60TianXinYongHuShouShuaRiQi:    row[15],
			TuiGuangRenYuanDaiMa:                         row[16],
		})
	}
	return phoneRows, nil
}

func ReadCallHistoryExcel(file *multipart.FileHeader) (map[string]CallHistoryRow, error) {
	callFile, err := file.Open()
	if err != nil {
		scrm.Logger().Error("open file error", err.Error())
		return nil, errors.New("文件异常")
	}
	defer func() {
		if err = callFile.Close(); err != nil {
			scrm.Logger().Error(err.Error())
		}
	}()
	f, err := excelize.OpenReader(callFile)
	if err != nil {
		scrm.Logger().Error(err.Error())
		return nil, err
	}
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		scrm.Logger().Error("no sheet")
		return nil, errors.New("表格式异常")
	}
	sheet := sheets[0]
	rows, err := f.GetRows(sheet)
	if err != nil {
		scrm.Logger().Error(err.Error())
		return nil, err
	}
	cols, err := f.GetCols(sheet)
	if err != nil {
		scrm.Logger().Error(err.Error())
		return nil, ErrSheetFormat
	}
	colLen := len(cols)
	callHistoryRowsMap := make(map[string]CallHistoryRow)
	for i := range rows {
		if i == 0 {
			continue
		}
		row := make([]string, colLen)
		for j := range row {
			val, err := f.GetCellValue(sheet, fmt.Sprintf("%s%d", util.ColMap[j+1], i+1))
			if err != nil {
				scrm.Logger().Error(err.Error())
				return nil, ErrSheetFormat
			}
			row[j] = val
		}
		if len(row) < 27 {
			scrm.Logger().Errorf("excel row length:%d", len(row))
			return nil, ErrSheetFormat
		}
		encryptPhone := strings.ToLower(hex.EncodeToString(sm3.Sm3Sum([]byte(row[3]))))
		callHistoryRowsMap[encryptPhone] = CallHistoryRow{
			Index:                 row[0],
			ID:                    row[1],
			OrderID:               row[2],
			Phone:                 row[3],
			Project:               row[4],
			CreatedAt:             row[5],
			Sentences:             row[6],
			ModelLabel:            row[7],
			CallLabel:             row[8],
			SeatLabel:             row[9],
			HangupLabel:           row[10],
			Comment:               row[11],
			DialUpCustomTime:      row[12],
			CustomAnswerTime:      row[13],
			SwitchSeatTime:        row[14],
			DialUpSeatTime:        row[15],
			SeatAnswerTime:        row[16],
			HangUpTime:            row[17],
			CustomRingingDuration: row[18],
			SeatRingingDuration:   row[19],
			AICallDuration:        row[20],
			SeatCallDuration:      row[21],
			SwitchingDuration:     row[22],
			TotalCallDuration:     row[23],
			SeatUserName:          row[24],
			SeatName:              row[25],
			Line:                  row[26],
		}
	}
	return callHistoryRowsMap, nil
}

func phoneRowToSlice(exportRows []ExportRow) [][]interface{} {
	var res [][]interface{}
	for _, row := range exportRows {
		s := []interface{}{
			row.WangZhanDaiMa,
			row.KeHuXinMing,
			row.ZhengJianHaoMa,
			row.ShouJiHaoMa,
			row.LuRuShiJian,
			row.ChuShenJieGuoDaiMa,
			row.ChuShenJieShuRiQi,
			row.ShenPiJueDingBiaoShi,
			row.ShenQingJueDingWanChengRiQi,
			row.DangYuePiHeDangYueZhuXiaoBiaoZhi,
			row.XinYongHuBiaoZhi,
			row.YouXiaoKeHuBiaoZhi,
			row.KaPianShouShuaRiQi,
			row.YouXiaoChuShen180TianBiaoShi,
			row.YouXiaoChuShen60TianXinYongHuShouShuaBiaoShi,
			row.YouXiaoChuShen60TianXinYongHuShouShuaRiQi,
			row.TuiGuangRenYuanDaiMa,
			func() string {
				if row.Match {
					return "1"
				}
				return ""
			}(),
			row.Index,
			row.ID,
			row.OrderID,
			row.Phone,
			row.Project,
			row.CreatedAt,
			row.Sentences,
			row.ModelLabel,
			row.CallLabel,
			row.SeatLabel,
			row.HangupLabel,
			row.Comment,
			row.DialUpCustomTime,
			row.CustomAnswerTime,
			row.SwitchSeatTime,
			row.DialUpSeatTime,
			row.SeatAnswerTime,
			row.HangUpTime,
			row.CustomRingingDuration,
			row.SeatRingingDuration,
			row.AICallDuration,
			row.SeatCallDuration,
			row.SwitchingDuration,
			row.TotalCallDuration,
			row.SeatUserName,
			row.SeatName,
			row.Line,
		}
		res = append(res, s)
	}
	return res
}
