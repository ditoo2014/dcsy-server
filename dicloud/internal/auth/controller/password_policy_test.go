package controller

import (
	"errors"
	"testing"

	"dicloud/models/model"
)

func TestValidatePasswordPolicy(t *testing.T) {
	user := model.User{
		Username: "testuser",
		TrueName: "张三",
		Email:    "testuser@example.com",
		Phone:    "13183639612",
		Renter:   "acmecloud",
	}

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "valid minimum length", password: "Qa1!bXeK"},
		{name: "valid maximum length", password: "Qa1!bXeK7@mNpRt"},
		{name: "too short", password: "Qa1!bX", wantErr: true},
		{name: "too long", password: "Qa1!bXeK7@mNpRtZ", wantErr: true},
		{name: "missing special", password: "Qa12bXeK", wantErr: true},
		{name: "contains whitespace", password: "Qa1! bXeK", wantErr: true},
		{name: "contains non ascii", password: "Qa1!密码bX", wantErr: true},
		{name: "repeated rune", password: "Qaaa1!Xe", wantErr: true},
		{name: "ascending digits", password: "Qa123!Xe", wantErr: true},
		{name: "descending digits", password: "Qa321!Xe", wantErr: true},
		{name: "ascending letters", password: "Qabc1!Xe", wantErr: true},
		{name: "mixed case ascending letters", password: "QaBc1!Xe", wantErr: true},
		{name: "descending letters", password: "Qcba1!Xe", wantErr: true},
		{name: "repeated digit block", password: "Qa1212!X", wantErr: true},
		{name: "paired digit sequence", password: "Qa112233!X", wantErr: true},
		{name: "contains username", password: "Testuser1!X", wantErr: true},
		{name: "contains real name", password: "Qa张三1!bX", wantErr: true},
		{name: "contains real name pinyin", password: "Zhangsan9!X", wantErr: true},
		{name: "contains real name initials", password: "Qa1!zsXe", wantErr: true},
		{name: "contains renter name", password: "Acmecloud9!X", wantErr: true},
		{name: "contains company token", password: "Ditoo9!bX", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePasswordPolicy(tt.password, user)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if !errors.Is(err, errPasswordPolicyViolation) {
				t.Fatalf("expected password policy violation, got %v", err)
			}
			if err.Error() != passwordPolicyMessage {
				t.Fatalf("expected unified message %q, got %q", passwordPolicyMessage, err.Error())
			}
		})
	}
}

func TestHasSequentialCharacterRun(t *testing.T) {
	tests := []struct {
		password string
		want     bool
	}{
		{password: "Xy123!aB", want: true},
		{password: "Xy321!aB", want: true},
		{password: "XaBc1!yZ", want: true},
		{password: "XcBa1!yZ", want: true},
		{password: "XaBd1!yZ", want: false},
	}
	for _, tt := range tests {
		if got := hasSequentialCharacterRun(tt.password, 3); got != tt.want {
			t.Fatalf("hasSequentialCharacterRun(%q) = %v, want %v", tt.password, got, tt.want)
		}
	}
}
