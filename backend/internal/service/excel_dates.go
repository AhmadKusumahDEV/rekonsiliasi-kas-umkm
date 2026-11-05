package service

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

var dimensionRowPattern = regexp.MustCompile("[0-9]+$")

func dimensionRows(dimension string) int {
	match := dimensionRowPattern.FindString(strings.TrimSpace(dimension))
	if match == "" {
		return 0
	}
	rows, err := strconv.Atoi(match)
	if err != nil {
		return 0
	}
	return rows
}

func isDateStyle(style *excelize.Style) bool {
	if style == nil {
		return false
	}
	if (style.NumFmt >= 14 && style.NumFmt <= 22) || (style.NumFmt >= 45 && style.NumFmt <= 47) {
		return true
	}
	if style.CustomNumFmt == nil {
		return false
	}
	format := strings.ToLower(*style.CustomNumFmt)
	return strings.Contains(format, "yy") || strings.Contains(format, "dd") || strings.Contains(format, "mm")
}

func normalizeExcelDates(workbook *excelize.File, sheet string, rows [][]string, date1904 bool) {
	for rowIndex, row := range rows {
		for colIndex, value := range row {
			if strings.TrimSpace(value) == "" {
				continue
			}
			cell, err := excelize.CoordinatesToCellName(colIndex+1, rowIndex+1)
			if err != nil {
				continue
			}
			styleID, err := workbook.GetCellStyle(sheet, cell)
			if err != nil {
				continue
			}
			style, err := workbook.GetStyle(styleID)
			if err != nil || !isDateStyle(style) {
				continue
			}
			serial, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if err != nil {
				continue
			}
			parsed, err := excelize.ExcelDateToTime(serial, date1904)
			if err != nil {
				continue
			}
			rows[rowIndex][colIndex] = parsed.Format("2006-01-02")
		}
	}
}
