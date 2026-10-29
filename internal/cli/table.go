package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

func (a *app) emitTable(raw json.RawMessage) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var data any
	if err := dec.Decode(&data); err != nil {
		return runtimeErr(err)
	}
	rows := flattenRows(data)
	if len(rows) == 0 {
		fmt.Fprintln(a.out, "(no rows)")
		return nil
	}
	cols := tableColumns(rows)
	w := tabwriter.NewWriter(a.out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(cols, "\t"))
	for _, row := range rows {
		line := make([]string, len(cols))
		for i, c := range cols {
			line[i] = cellString(row[c])
		}
		fmt.Fprintln(w, strings.Join(line, "\t"))
	}
	return w.Flush()
}

func flattenRows(data any) []map[string]any {
	switch t := data.(type) {
	case []any:
		out := make([]map[string]any, 0, len(t))
		for _, item := range t {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	case map[string]any:
		if r, ok := t["result"]; ok {
			return flattenRows(r)
		}
		return []map[string]any{t}
	default:
		return nil
	}
}

func tableColumns(rows []map[string]any) []string {
	priority := []string{"id", "fingerprint", "name", "status", "severity", "workflow_id", "provider_id", "provider_type"}
	seen := map[string]bool{}
	var cols []string
	for _, p := range priority {
		for _, row := range rows {
			if _, ok := row[p]; ok && !seen[p] {
				cols = append(cols, p)
				seen[p] = true
			}
		}
	}
	for _, row := range rows {
		for k := range row {
			if !seen[k] && len(cols) < 8 {
				cols = append(cols, k)
				seen[k] = true
			}
		}
	}
	if len(cols) == 0 {
		cols = []string{"value"}
	}
	return cols
}

func cellString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		if len(t) > 80 {
			return t[:77] + "..."
		}
		return t
	case json.Number, float64, int, int64, bool:
		return fmt.Sprint(t)
	default:
		b, _ := json.Marshal(t)
		s := string(b)
		if len(s) > 80 {
			return s[:77] + "..."
		}
		return s
	}
}

func readBodyFlag(jsonStr, file string) ([]byte, error) {
	if jsonStr != "" {
		if !json.Valid([]byte(jsonStr)) {
			return nil, usageErr("--json must be valid JSON")
		}
		return []byte(jsonStr), nil
	}
	if file != "" {
		p := strings.TrimPrefix(file, "@")
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, runtimeErr(err)
		}
		if !json.Valid(b) {
			return nil, usageErr("body file must contain valid JSON")
		}
		return b, nil
	}
	return nil, nil
}
