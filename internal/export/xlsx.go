package exporter

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

func WriteXLSX(w io.Writer, releaseExport ReleaseExport) error {
	file, err := buildWorkbook(releaseExport)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteTo(w)
	return err
}

func WriteXLSXFile(path string, releaseExport ReleaseExport) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := buildWorkbook(releaseExport)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.SaveAs(path)
}

func buildWorkbook(releaseExport ReleaseExport) (*excelize.File, error) {
	file := excelize.NewFile()
	headerStyle, err := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"EDEFF3"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "bottom", Color: "D0D7DE", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	for index, sheet := range releaseExport.Sheets {
		if sheet.Name == "" {
			return nil, fmt.Errorf("sheet name is required")
		}
		if index == 0 {
			if err := file.SetSheetName("Sheet1", sheet.Name); err != nil {
				return nil, err
			}
		} else if _, err := file.NewSheet(sheet.Name); err != nil {
			return nil, err
		}
		if err := writeSheet(file, sheet, headerStyle); err != nil {
			return nil, err
		}
	}
	if len(releaseExport.Sheets) > 0 {
		if index, err := file.GetSheetIndex(releaseExport.Sheets[0].Name); err == nil {
			file.SetActiveSheet(index)
		}
	}
	return file, nil
}

func writeSheet(file *excelize.File, sheet Sheet, headerStyle int) error {
	if len(sheet.Headers) == 0 {
		return fmt.Errorf("sheet %q must include headers", sheet.Name)
	}
	if err := setRow(file, sheet.Name, 1, sheet.Headers); err != nil {
		return err
	}
	lastCol, err := excelize.ColumnNumberToName(len(sheet.Headers))
	if err != nil {
		return err
	}
	headerRange := "A1:" + lastCol + "1"
	if err := file.SetCellStyle(sheet.Name, "A1", lastCol+"1", headerStyle); err != nil {
		return err
	}
	if err := file.SetPanes(sheet.Name, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		return err
	}

	widths := make([]float64, len(sheet.Headers))
	for i, header := range sheet.Headers {
		widths[i] = boundedWidth(header)
	}
	for rowIndex, row := range sheet.Rows {
		if err := setRow(file, sheet.Name, rowIndex+2, row); err != nil {
			return err
		}
		for i, value := range row {
			if i >= len(widths) {
				break
			}
			widths[i] = math.Max(widths[i], boundedWidth(value))
		}
	}
	for i, width := range widths {
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}
		if err := file.SetColWidth(sheet.Name, col, col, width); err != nil {
			return err
		}
	}
	lastRow := len(sheet.Rows) + 1
	filterRange := fmt.Sprintf("A1:%s%d", lastCol, lastRow)
	if lastRow < 1 {
		filterRange = headerRange
	}
	_ = file.AutoFilter(sheet.Name, filterRange, []excelize.AutoFilterOptions{})
	return nil
}

func setRow(file *excelize.File, sheet string, row int, values []string) error {
	record := make([]interface{}, len(values))
	for i, value := range values {
		record[i] = SafeCellString(value)
	}
	cell, err := excelize.CoordinatesToCellName(1, row)
	if err != nil {
		return err
	}
	return file.SetSheetRow(sheet, cell, &record)
}

func boundedWidth(value string) float64 {
	length := float64(len([]rune(value)) + 2)
	if length < 12 {
		return 12
	}
	if length > 42 {
		return 42
	}
	return length
}
