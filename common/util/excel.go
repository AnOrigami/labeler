package util

import (
	"context"
	"embed"
	"encoding/base64"
	"time"

	"github.com/xuri/excelize/v2"

	"go-admin/common/log"
)

func CreateExcelFile(data [][]interface{}, columns []string, model string) (*string, string, error) {
	filename := model + GetExportId() + ".xlsx"
	s, err := NewExcel(columns, data)
	if err != nil {
		return nil, "", err
	}
	return s, filename, nil
}

func GetExportId() string {
	return time.Now().UTC().Add(8 * time.Hour).Format("20060102")
}

func NewExcel(columnNames []string, values [][]interface{}) (*string, error) {
	f := excelize.NewFile()
	for i, columnName := range columnNames {
		aixs, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue("Sheet1", aixs, columnName)
	}
	for row, rowValues := range values {
		for col, val := range rowValues {
			aixs, _ := excelize.CoordinatesToCellName(col+1, row+2)
			_ = f.SetCellValue("Sheet1", aixs, val)
		}
	}
	//if err := f.SaveAs("测试.xlsx"); err != nil {
	//	fmt.Println(err)
	//}
	buf, _ := f.WriteToBuffer()
	result := base64.StdEncoding.EncodeToString(buf.Bytes())
	return &result, nil
}

func MakeExcelFromData(data [][]interface{}, columns []string) *excelize.File {
	f := excelize.NewFile()
	for i, columnName := range columns {
		aixs, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue("Sheet1", aixs, columnName)
	}
	for row, rowValues := range data {
		for col, val := range rowValues {
			aixs, _ := excelize.CoordinatesToCellName(col+1, row+2)
			_ = f.SetCellValue("Sheet1", aixs, val)
		}
	}
	return f
}

func GetExcelFileName(model string) string {
	return model + time.Now().UTC().Add(8*time.Hour).Format("20060102") + ".xlsx"
}

//go:embed task5score.xlsx
var task5excel embed.FS

func EmbedExcelData(excelName string, data [][]interface{}, ctx context.Context) (*string, string, error) {
	var result string
	excelName = excelName + ".xlsx"
	r, err := task5excel.Open("task5score.xlsx")
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return &result, excelName, err
	}

	f, err := excelize.OpenReader(r)
	defer func() {
		// Close the spreadsheet.
		if err := f.Close(); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return
		}
	}()
	if err != nil {
		panic(err)
	}
	//if err := f.SetSheetRow(f.GetSheetName(0), "A3", &[]any{"a", "b", "c"}); err != nil {
	//	panic(err)
	//}
	for row, rowValues := range data {
		aixs, _ := excelize.CoordinatesToCellName(1, row+3)
		err := f.SetSheetRow("Sheet1", aixs, &rowValues)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return &result, excelName, err
		}
	}

	//if err := f.SaveAs("测试.xlsx"); err != nil {
	//	fmt.Println(err)
	//}

	//f.SaveAs(excelName)
	buf, _ := f.WriteToBuffer()
	result = base64.StdEncoding.EncodeToString(buf.Bytes())
	return &result, excelName, nil

}

var ColMap = map[int]string{
	1:  "A",
	2:  "B",
	3:  "C",
	4:  "D",
	5:  "E",
	6:  "F",
	7:  "G",
	8:  "H",
	9:  "I",
	10: "J",
	11: "K",
	12: "L",
	13: "M",
	14: "N",
	15: "O",
	16: "P",
	17: "Q",
	18: "R",
	19: "S",
	20: "T",
	21: "U",
	22: "V",
	23: "W",
	24: "X",
	25: "Y",
	26: "Z",
	27: "AA",
}
