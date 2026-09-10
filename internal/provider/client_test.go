package provider

import (
	"errors"
	"testing"

	"google.golang.org/api/googleapi"
)

func TestParentOfName(t *testing.T) {
	cases := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{name: "properties/123/dataStreams/456", want: "properties/123"},
		{name: "properties/123/dataStreams/456/measurementProtocolSecrets/789", want: "properties/123/dataStreams/456"},
		{name: "properties/123/customDimensions/456", want: "properties/123"},
		// 親を持たない（アカウント直下のプロパティ）は Parent フィールドを使うので、ここでは扱わない
		{name: "properties/123", wantErr: true},
		{name: "properties/123/dataStreams", wantErr: true},
		{name: "", wantErr: true},
	}

	for _, c := range cases {
		got, err := parentOfName(c.name)
		if c.wantErr {
			if err == nil {
				t.Errorf("parentOfName(%q) はエラーになるべき（got %q）", c.name, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parentOfName(%q): %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("parentOfName(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestIsNotFound(t *testing.T) {
	if !isNotFound(&googleapi.Error{Code: 404}) {
		t.Error("404 を not found と判定していない")
	}
	if isNotFound(&googleapi.Error{Code: 403}) {
		t.Error("403 を not found と判定している")
	}
	if isNotFound(errors.New("network")) {
		t.Error("API 以外のエラーを not found と判定している")
	}
	if !isNotFound(errors.Join(errors.New("wrapped"), &googleapi.Error{Code: 404})) {
		t.Error("ラップされた 404 を判定できていない")
	}
}

func TestNewAdminServiceRejectsMissingCredentialsFile(t *testing.T) {
	_, err := newAdminService(t.Context(), "/path/that/does/not/exist.json")
	if err == nil {
		t.Error("存在しないファイルを指定してもエラーにならない")
	}
}
