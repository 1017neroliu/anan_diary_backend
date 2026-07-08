package models

import (
	"testing"
)

func TestGrowthRecord_GetTags(t *testing.T) {
	tests := []struct {
		name string
		tags string
		want []string
	}{
		{"空字符串应返回空切片", "", []string{}},
		{"空数组字面量应返回空切片", "[]", []string{}},
		{"单标签", `["满月"]`, []string{"满月"}},
		{"多标签", `["满月","百日","第一次笑"]`, []string{"满月", "百日", "第一次笑"}},
		{"无效 JSON 应返回空切片", "not-json", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &GrowthRecord{Tags: tt.tags}
			got := r.GetTags()
			if len(got) != len(tt.want) {
				t.Errorf("GetTags() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("GetTags()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGrowthRecord_TableName(t *testing.T) {
	r := GrowthRecord{}
	if got := r.TableName(); got != "growth_records" {
		t.Errorf("TableName() = %q, want %q", got, "growth_records")
	}
}