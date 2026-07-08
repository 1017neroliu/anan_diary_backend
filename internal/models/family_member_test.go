package models

import "testing"

func TestFamilyMember_GetFreeLimit(t *testing.T) {
	fm := FamilyMember{}
	if got := fm.GetFreeLimit(); got != 2 {
		t.Errorf("GetFreeLimit() = %d, want 2", got)
	}
}

func TestFamilyMember_TableName(t *testing.T) {
	fm := FamilyMember{}
	if got := fm.TableName(); got != "family_members" {
		t.Errorf("TableName() = %q, want %q", got, "family_members")
	}
}