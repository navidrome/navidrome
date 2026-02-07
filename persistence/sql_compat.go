package persistence

import (
	"fmt"

	"github.com/navidrome/navidrome/db"
)

// SQL compatibility helpers for SQLite and PostgreSQL

// JsonParticipantExists checks if a participant role contains a value.
// Returns (from clause, condition column name)
func JsonParticipantExists(role string, valueKey string) (from string, cond string) {
	if db.IsPostgres() {
		return fmt.Sprintf("jsonb_array_elements(participants::jsonb->'%s') as elem", role),
			fmt.Sprintf("elem->>'%s'", valueKey)
	}
	return fmt.Sprintf("json_tree(participants, '$.%s')", role), "value"
}

// JsonTagExists checks if a tag array contains a value.
// Returns (from clause, condition column name)
func JsonTagExists(tag string) (from string, cond string) {
	if db.IsPostgres() {
		return fmt.Sprintf("jsonb_array_elements_text(tags::jsonb->'%s') as elem", tag), "elem"
	}
	return fmt.Sprintf("json_tree(tags, '$.%s')", tag), "value"
}

func JsonArrayLength(col, path string) string {
	if db.IsPostgres() {
		return fmt.Sprintf("jsonb_array_length(%s::jsonb->'%s')", col, path)
	}
	return fmt.Sprintf("json_array_length(%s, '$.%s')", col, path)
}

func JsonGroupObject(keyExpr, valueExpr string) string {
	if db.IsPostgres() {
		return fmt.Sprintf("jsonb_object_agg(%s, %s)", keyExpr, valueExpr)
	}
	return fmt.Sprintf("json_group_object(%s, %s)", keyExpr, valueExpr)
}

func JsonGroupArray(expr string) string {
	if db.IsPostgres() {
		return fmt.Sprintf("json_agg(%s)", expr)
	}
	return fmt.Sprintf("json_group_array(%s)", expr)
}

func JsonExtract(col, path string) string {
	if db.IsPostgres() {
		return fmt.Sprintf("%s::jsonb->>'%s'", col, path)
	}
	return fmt.Sprintf("json_extract(%s, '$.%s')", col, path)
}

func JsonObject(pairs ...string) string {
	if db.IsPostgres() {
		return fmt.Sprintf("jsonb_build_object(%s)", joinPairs(pairs))
	}
	return fmt.Sprintf("json_object(%s)", joinPairs(pairs))
}

func BoolValue(b bool) string {
	if db.IsPostgres() {
		if b {
			return "true"
		}
		return "false"
	}
	if b {
		return "1"
	}
	return "0"
}

// CoalesceBool wraps coalesce(); PostgreSQL needs bool literals, not 0/1.
func CoalesceBool(col string, defaultVal bool) string {
	return fmt.Sprintf("coalesce(%s, %s)", col, BoolValue(defaultVal))
}

// UserTable quotes the table name; "user" is reserved in PostgreSQL.
func UserTable() string {
	if db.IsPostgres() {
		return `"user"`
	}
	return "user"
}

func UserColumn(col string) string {
	return UserTable() + "." + col
}

func joinPairs(pairs []string) string {
	result := ""
	for i, p := range pairs {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}
