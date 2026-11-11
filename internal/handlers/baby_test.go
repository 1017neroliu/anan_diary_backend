package handlers

import "testing"

func TestParseBirthDate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		// RFC3339 带时区：按原时区的日期部分解析
		{"RFC3339 带东八区", "2025-06-15T10:30:00+08:00", "2025-06-15", true},
		{"RFC3339 带UTC", "2025-06-15T23:30:00Z", "2025-06-15", true},
		// 无时区 ISO
		{"无时区ISO", "2025-06-15T10:30:00", "2025-06-15", true},
		// 纯日期
		{"纯日期", "2025-06-15", "2025-06-15", true},
		// 非法格式
		{"斜杠分隔", "2025/06/15", "", false},
		{"非补零日期", "2025-6-5", "", false},
		{"非日期字符串", "abc", "", false},
		{"空串", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBirthDate(tt.input)
			if tt.ok {
				if err != nil {
					t.Fatalf("parseBirthDate(%q) 意外失败: %v", tt.input, err)
				}
				if got != tt.want {
					t.Errorf("parseBirthDate(%q) = %q, 期望 %q", tt.input, got, tt.want)
				}
			} else {
				if err == nil {
					t.Errorf("parseBirthDate(%q) = %q, 期望失败", tt.input, got)
				}
			}
		})
	}
}
