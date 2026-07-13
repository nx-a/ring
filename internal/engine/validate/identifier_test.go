package validate

import "testing"

func TestIsIdentifier(t *testing.T) {
	cases := []struct {
		in    string
		valid bool
	}{
		{"bucket_1", true},
		{"BucketName", true},
		{"_bucket", true},
		{"1bucket", false},
		{"bucket-name", false},
		{"select", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := IsIdentifier(tc.in); got != tc.valid {
			t.Errorf("IsIdentifier(%q) = %v, want %v", tc.in, got, tc.valid)
		}
	}
}

func TestBucketName(t *testing.T) {
	if err := BucketName("valid_bucket"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := BucketName("drop"); err == nil {
		t.Error("expected error for SQL keyword")
	}
	if err := BucketName(""); err == nil {
		t.Error("expected error for empty name")
	}
}
