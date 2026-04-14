package db

import (
	"testing"
)

func TestSplitStatements(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "empty",
			input: "",
			want:  0,
		},
		{
			name:  "single statement",
			input: "CREATE TABLE foo (id INT);",
			want:  1,
		},
		{
			name:  "two statements",
			input: "CREATE TABLE foo (id INT); CREATE TABLE bar (id INT);",
			want:  2,
		},
		{
			name:  "whitespace only",
			input: "   \n\t  ",
			want:  0,
		},
		{
			name:  "trailing newlines after semicolons",
			input: "SELECT 1;\n\nSELECT 2;\n\n",
			want:  2,
		},
		{
			name:  "no trailing semicolon",
			input: "SELECT 1; SELECT 2",
			want:  2,
		},
		{
			name:  "semicolons with only whitespace between",
			input: ";\n;\n;",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitStatements(tt.input)
			if len(got) != tt.want {
				t.Errorf("splitStatements(%q) returned %d statements, want %d\n  got: %v", tt.input, len(got), tt.want, got)
			}
		})
	}
}

func TestSplitStatements_MysqlSchema(t *testing.T) {
	// Verify the actual mysqlSchema splits into the expected number of CREATE statements.
	stmts := splitStatements(mysqlSchema)
	if len(stmts) < 4 {
		t.Errorf("mysqlSchema split into %d statements, expected at least 4 (users, task_lists, list_members, tasks, task_updates)", len(stmts))
	}
	// Each statement should be non-empty and end with a semicolon.
	for i, s := range stmts {
		if s == "" {
			t.Errorf("statement %d is empty", i)
		}
		if s[len(s)-1] != ';' {
			t.Errorf("statement %d does not end with semicolon: %q", i, s)
		}
	}
}

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"", ""},
		{"  hello  ", "hello"},
		{"\n\t world \r\n", "world"},
		{"abc", "abc"},
	}
	for _, tt := range tests {
		got := trimSpace(tt.input)
		if got != tt.want {
			t.Errorf("trimSpace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
