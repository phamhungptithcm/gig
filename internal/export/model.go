package exporter

import (
	"strings"
	"time"
)

type Format string

const (
	FormatHuman    Format = "human"
	FormatMarkdown Format = "markdown"
	FormatJSON     Format = "json"
	FormatXLSX     Format = "xlsx"
	FormatCSV      Format = "csv"
)

type TargetKind string

const (
	TargetStdout    TargetKind = "stdout"
	TargetFile      TargetKind = "file"
	TargetDirectory TargetKind = "directory"
)

type ResolvedOutput struct {
	Format Format
	Target TargetKind
	Path   string
}

type Field struct {
	Name  string
	Value string
}

type Sheet struct {
	Name    string
	CSVName string
	Headers []string
	Rows    [][]string
}

type ReleaseExport struct {
	Sheets    []Sheet
	SingleCSV Sheet
}

type Options struct {
	GeneratedAt       time.Time
	GeneratedBy       string
	Command           string
	ScopeLabel        string
	Mode              string
	Provider          string
	ConfigPath        string
	AuthSource        string
	WorkingDirectory  string
	JSONSchemaVersion string
	ToolVersions      map[string]string
	DataIncomplete    bool
	IncompleteReason  string
}

func (o Options) generatedAt() time.Time {
	if o.GeneratedAt.IsZero() {
		return time.Now()
	}
	return o.GeneratedAt
}

func (o Options) generatedAtString() string {
	return o.generatedAt().Format(time.RFC3339)
}

func (o Options) timezone() string {
	zone, offset := o.generatedAt().Zone()
	if zone == "" {
		return "UTC"
	}
	hours := offset / 3600
	minutes := (offset % 3600) / 60
	sign := "+"
	if offset < 0 {
		sign = "-"
		if hours < 0 {
			hours = -hours
		}
		if minutes < 0 {
			minutes = -minutes
		}
	}
	return strings.TrimSpace(zone + " UTC" + sign + twoDigit(hours) + ":" + twoDigit(minutes))
}

func twoDigit(value int) string {
	if value < 10 {
		return "0" + string(rune('0'+value))
	}
	return string(rune('0'+(value/10))) + string(rune('0'+(value%10)))
}

func fieldSheet(name, csvName string, fields []Field) Sheet {
	rows := make([][]string, 0, len(fields))
	for _, field := range fields {
		rows = append(rows, []string{field.Name, field.Value})
	}
	return Sheet{
		Name:    name,
		CSVName: csvName,
		Headers: []string{"Field", "Value"},
		Rows:    rows,
	}
}

func SafeCellString(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	switch value[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}
