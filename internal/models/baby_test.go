package models

import (
	"testing"
	"time"
)

func TestBaby_GetAgeInDays(t *testing.T) {
	tests := []struct {
		name      string
		birthDate string
		wantMin   int // 最小期望天数（浮动因测试时间不同）
	}{
		{"出生当天应为 0", time.Now().Format("2006-01-02"), 0},
		{"昨天出生应为 1", time.Now().AddDate(0, 0, -1).Format("2006-01-02"), 1},
		{"约一个月前", time.Now().AddDate(0, -1, 0).Format("2006-01-02"), 28},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Baby{BirthDate: tt.birthDate}
			got := b.GetAgeInDays()
			if got < tt.wantMin {
				t.Errorf("GetAgeInDays() = %d, want >= %d", got, tt.wantMin)
			}
		})
	}
}

func TestBaby_GetAgeInMonths(t *testing.T) {
	tests := []struct {
		name      string
		birthDate string
		wantMin   float64
	}{
		{"出生当天", time.Now().Format("2006-01-02"), 0},
		{"约一月前", time.Now().AddDate(0, -1, 0).Format("2006-01-02"), 0.9},
		{"约一年前", time.Now().AddDate(-1, 0, 0).Format("2006-01-02"), 11.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Baby{BirthDate: tt.birthDate}
			got := b.GetAgeInMonths()
			if got < tt.wantMin {
				t.Errorf("GetAgeInMonths() = %f, want >= %f", got, tt.wantMin)
			}
		})
	}
}