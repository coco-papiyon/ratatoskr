package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/itchyny/gojq"
	"gopkg.in/yaml.v3"
)

func DefaultStructuredTableRules() []StructuredTableRule {
	defaultJQ := `if type == "array" then . elif type == "object" and (.items | type == "array") then .items elif type == "object" then [.] else [] end`
	return []StructuredTableRule{
		{Name: "JSON objects", FilePattern: `(?i).*\.json$`, JQ: defaultJQ},
		{Name: "YAML objects", FilePattern: `(?i).*\.(yaml|yml)$`, JQ: defaultJQ},
	}
}

func (a *App) GetStructuredTableRules() []StructuredTableRule {
	return append([]StructuredTableRule(nil), a.structuredTableRules...)
}

func (a *App) UpdateStructuredTableRules(rules []StructuredTableRule) error {
	if err := validateStructuredTableRules(rules); err != nil {
		return err
	}
	contents, err := yaml.Marshal(rules)
	if err != nil {
		return fmt.Errorf("表変換設定をYAMLに変換できません: %w", err)
	}
	path, err := structuredRulesPath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		return fmt.Errorf("表変換設定を保存できません: %w", err)
	}
	a.structuredTableRules = append([]StructuredTableRule(nil), rules...)
	return nil
}

func validateStructuredTableRules(rules []StructuredTableRule) error {
	if len(rules) == 0 {
		return fmt.Errorf("表変換ルールを1件以上指定してください")
	}
	for index, rule := range rules {
		if strings.TrimSpace(rule.Name) == "" || strings.TrimSpace(rule.FilePattern) == "" || strings.TrimSpace(rule.JQ) == "" {
			return fmt.Errorf("表変換ルール %d の名前、対象ファイル、jqを入力してください", index+1)
		}
		if _, err := regexp.Compile(rule.FilePattern); err != nil {
			return fmt.Errorf("表変換ルール %s の正規表現が不正です: %w", rule.Name, err)
		}
		if _, err := gojq.Parse(rule.JQ); err != nil {
			return fmt.Errorf("表変換ルール %s のjqが不正です: %w", rule.Name, err)
		}
	}
	return nil
}

func (a *App) ConvertStructuredToTable(filePath, content string) (*StructuredTable, error) {
	for _, rule := range a.structuredTableRules {
		pattern, err := regexp.Compile(rule.FilePattern)
		if err != nil {
			return nil, fmt.Errorf("変換ルール %s の正規表現が不正です: %w", rule.Name, err)
		}
		if !pattern.MatchString(filePath) && !pattern.MatchString(filepath.Base(filePath)) {
			continue
		}
		var input any
		if strings.EqualFold(filepath.Ext(filePath), ".yaml") || strings.EqualFold(filepath.Ext(filePath), ".yml") {
			if err := yaml.Unmarshal([]byte(content), &input); err != nil {
				return nil, fmt.Errorf("YAML を解析できません: %w", err)
			}
			input = normalizeYAMLValue(input)
		} else if err := json.Unmarshal([]byte(content), &input); err != nil {
			return nil, fmt.Errorf("JSON を解析できません: %w", err)
		}
		query, err := gojq.Parse(rule.JQ)
		if err != nil {
			return nil, fmt.Errorf("変換ルール %s の jq が不正です: %w", rule.Name, err)
		}
		values := make([]any, 0)
		iterator := query.Run(input)
		for {
			value, ok := iterator.Next()
			if !ok {
				break
			}
			if queryErr, ok := value.(error); ok {
				return nil, fmt.Errorf("変換ルール %s の実行に失敗しました: %w", rule.Name, queryErr)
			}
			if array, ok := value.([]any); ok {
				values = append(values, array...)
			} else {
				values = append(values, value)
			}
		}
		return makeStructuredTable(rule.Name, values), nil
	}
	return nil, nil
}

func makeStructuredTable(ruleName string, values []any) *StructuredTable {
	if table, ok := makeArrayTable(ruleName, values); ok {
		return table
	}
	columnSet := map[string]bool{}
	objects := make([]map[string]any, 0, len(values))
	for _, value := range values {
		object, ok := value.(map[string]any)
		if !ok {
			object = map[string]any{"value": value}
		}
		objects = append(objects, object)
		for key := range object {
			columnSet[key] = true
		}
	}
	columns := make([]string, 0, len(columnSet))
	for column := range columnSet {
		columns = append(columns, column)
	}
	sort.Strings(columns)
	rows := make([][]string, 0, len(objects))
	for _, object := range objects {
		row := make([]string, len(columns))
		for index, column := range columns {
			row[index] = structuredCellText(object[column])
		}
		rows = append(rows, row)
	}
	return &StructuredTable{RuleName: ruleName, Columns: columns, Rows: rows}
}

func makeArrayTable(ruleName string, values []any) (*StructuredTable, bool) {
	if len(values) == 0 {
		return nil, false
	}
	headers, ok := values[0].([]any)
	if !ok || len(headers) == 0 {
		return nil, false
	}
	columns := make([]string, len(headers))
	for index, header := range headers {
		columns[index] = structuredCellText(header)
	}
	rows := make([][]string, 0, len(values)-1)
	for _, value := range values[1:] {
		cells, ok := value.([]any)
		if !ok {
			return nil, false
		}
		row := make([]string, len(columns))
		for index := range columns {
			if index < len(cells) {
				row[index] = structuredCellText(cells[index])
			}
		}
		rows = append(rows, row)
	}
	return &StructuredTable{RuleName: ruleName, Columns: columns, Rows: rows}, true
}

func structuredCellText(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(encoded)
}

func normalizeYAMLValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, child := range typed {
			result[key] = normalizeYAMLValue(child)
		}
		return result
	case map[any]any:
		result := make(map[string]any, len(typed))
		for key, child := range typed {
			result[fmt.Sprint(key)] = normalizeYAMLValue(child)
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for index, child := range typed {
			result[index] = normalizeYAMLValue(child)
		}
		return result
	default:
		return value
	}
}

func loadStructuredTableRules() ([]StructuredTableRule, error) {
	path, err := structuredRulesPath()
	if err != nil {
		return DefaultStructuredTableRules(), err
	}
	contents, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		rules := DefaultStructuredTableRules()
		encoded, encodeErr := yaml.Marshal(rules)
		if encodeErr != nil {
			return rules, encodeErr
		}
		if writeErr := os.WriteFile(path, encoded, 0o644); writeErr != nil {
			return rules, writeErr
		}
		return rules, nil
	}
	if err != nil {
		return DefaultStructuredTableRules(), err
	}
	var rules []StructuredTableRule
	if err := yaml.Unmarshal(contents, &rules); err != nil {
		return nil, fmt.Errorf("%s を読み込めません。配列形式で記載してください: %w", path, err)
	}
	validRules := make([]StructuredTableRule, 0, len(rules))
	for _, rule := range rules {
		if rule.FilePattern != "" && rule.JQ != "" {
			validRules = append(validRules, rule)
		}
	}
	if len(validRules) == 0 {
		return DefaultStructuredTableRules(), nil
	}
	return validRules, nil
}

func structuredRulesPath() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	directory := filepath.Join(workingDirectory, "config")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(directory, "json-table.yaml"), nil
}
