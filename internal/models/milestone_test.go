package models

import (
	"testing"
	"time"
)

func TestCheckMilestones(t *testing.T) {
	birthDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		recordDate time.Time
		wantTags   []string // 期望包含这些标签（可能有额外周标记，仅检查子集）
	}{
		{"出生当天（第0天）", birthDate, []string{}},
		{"满月（30天）", birthDate.AddDate(0, 0, 30), []string{"满月"}},
		{"百日（100天）", birthDate.AddDate(0, 0, 100), []string{"百日", "百日快乐"}},
		{"半岁（180天）", birthDate.AddDate(0, 0, 180), []string{"半岁"}},
		{"一周岁（365天）", birthDate.AddDate(0, 0, 365), []string{"一周岁"}},
		{"第一周（7天）", birthDate.AddDate(0, 0, 7), []string{"第一周"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckMilestones(birthDate, tt.recordDate)

			// 检查期望的标签是否都在结果中
			gotSet := make(map[string]bool, len(got))
			for _, g := range got {
				gotSet[g] = true
			}
			for _, want := range tt.wantTags {
				if !gotSet[want] {
					t.Errorf("CheckMilestones() = %v, 期望包含 %q", got, want)
				}
			}
		})
	}
}

func TestCheckMilestones_RecordBeforeBirth(t *testing.T) {
	birthDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	recordDate := birthDate.AddDate(0, 0, -5) // 出生前 5 天

	got := CheckMilestones(birthDate, recordDate)
	if len(got) != 0 {
		t.Errorf("出生前记录应无里程碑标签, got = %v", got)
	}
}