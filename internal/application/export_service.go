package application

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/config"
)

// MarkdownExporter exports tasks as a Markdown checklist.
type MarkdownExporter struct {
	Priorities []config.PriorityDef
}

// Export implements domain.Exporter for Markdown output.
func (e MarkdownExporter) Export(tasks []domain.Task) ([]byte, error) {
	var sb strings.Builder
	for _, t := range tasks {
		check := "[ ]"
		if t.Status == domain.StatusDone {
			check = "[x]"
		}
		priLabel := config.PriorityLabel(e.Priorities, int(t.Priority))
		line := fmt.Sprintf("- %s **%s** (%s)", check, t.Title, priLabel)
		if t.DueDate != nil {
			line += " due:" + t.DueDate.Format("2006-01-02")
		}
		sb.WriteString(line + "\n")
	}
	return []byte(sb.String()), nil
}

// CSVExporter exports tasks as comma-separated values.
type CSVExporter struct {
	Priorities []config.PriorityDef
}

// Export implements domain.Exporter for CSV output.
func (e CSVExporter) Export(tasks []domain.Task) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	_ = w.Write([]string{"id", "title", "status", "priority", "due_date", "tags", "list_id"})
	for _, t := range tasks {
		due := ""
		if t.DueDate != nil {
			due = t.DueDate.Format("2006-01-02")
		}
		priLabel := config.PriorityLabel(e.Priorities, int(t.Priority))
		_ = w.Write([]string{
			t.ID,
			t.Title,
			string(t.Status),
			priLabel,
			due,
			strings.Join(t.Tags, "|"),
			t.ListID,
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("csv export: %w", err)
	}
	return buf.Bytes(), nil
}
