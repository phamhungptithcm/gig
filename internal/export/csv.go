package exporter

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func WriteSingleCSV(w io.Writer, sheet Sheet) error {
	writer := csv.NewWriter(w)
	if err := writer.Write(safeRecord(sheet.Headers)); err != nil {
		return err
	}
	for _, row := range sheet.Rows {
		if err := writer.Write(safeRecord(row)); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func WriteSingleCSVFile(path string, sheet Sheet) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return WriteSingleCSV(file, sheet)
}

func WriteCSVDirectory(path string, releaseExport ReleaseExport) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("output directory is required")
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	for _, sheet := range releaseExport.Sheets {
		csvName := strings.TrimSpace(sheet.CSVName)
		if csvName == "" {
			csvName = csvFileName(sheet.Name)
		}
		if err := WriteSingleCSVFile(filepath.Join(path, csvName), sheet); err != nil {
			return err
		}
	}
	return nil
}

func safeRecord(record []string) []string {
	safe := make([]string, len(record))
	for i, value := range record {
		safe[i] = SafeCellString(value)
	}
	return safe
}

func csvFileName(sheetName string) string {
	name := strings.ToLower(strings.TrimSpace(sheetName))
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	if name == "" {
		name = "sheet"
	}
	return name + ".csv"
}
