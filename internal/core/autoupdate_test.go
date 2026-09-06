package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"v0.3.0", "v0.2.0", true},
		{"0.3.0", "0.2.0", true},
		{"v0.2.0", "v0.2.0", false},
		{"0.2.0", "0.2.0", false},
		{"v0.2.0", "v0.3.0", false},
		{"v0.1.9", "v0.2.0", false},
		{"v1.0.0", "v0.9.9", true},
		{"v0.2.1", "v0.2.0", true},
		{"v0.2.0", "v0.2.1", false},
		{"v0.2.0", "dev", true},
		{"v0.2.0", "", true},
		{"", "v0.2.0", false},
		{"", "", false},
		{"v1.0.0-rc1", "v0.9.0", true},
	}

	for _, tc := range tests {
		result := IsNewerVersion(tc.v1, tc.v2)
		if result != tc.expected {
			t.Errorf("IsNewerVersion(%q, %q) = %v; want %v", tc.v1, tc.v2, result, tc.expected)
		}
	}
}

func TestFetchLatestCliReleaseTag(t *testing.T) {
	// Test standard API response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name": "v0.3.0"}`))
	}))
	defer server.Close()

	ctx := context.Background()

	// Live GitHub fallback test (just verifies it doesn't crash)
	tag, err := FetchLatestCliReleaseTag(ctx)
	if err != nil {
		t.Logf("FetchLatestCliReleaseTag returned error (possibly offline or rate-limited): %v", err)
	} else if tag == "" {
		t.Errorf("FetchLatestCliReleaseTag returned empty tag")
	} else {
		t.Logf("FetchLatestCliReleaseTag returned: %s", tag)
	}
}
