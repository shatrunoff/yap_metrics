package utils

import (
	"testing"
)

func TestComputeHMACSHA256(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		key  string
	}{
		{
			name: "simple case",
			data: []byte("hello"),
			key:  "secret",
		},
		{
			name: "empty data",
			data: []byte(""),
			key:  "secret",
		},
		{
			name: "empty key",
			data: []byte("hello"),
			key:  "",
		},
		{
			name: "json data",
			data: []byte(`{"test":"data"}`),
			key:  "my-secret-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeHMACSHA256(tt.data, tt.key)
			if result == "" {
				t.Error("ComputeHMACSHA256() returned empty string")
			}
			if len(result) != 64 {
				t.Errorf("ComputeHMACSHA256() returned hash with length %d, want 64", len(result))
			}
		})
	}
}

func TestComputeHMACSHA256_Consistency(t *testing.T) {
	data := []byte("test data")
	key := "test key"

	result1 := ComputeHMACSHA256(data, key)
	result2 := ComputeHMACSHA256(data, key)

	if result1 != result2 {
		t.Error("ComputeHMACSHA256 should return consistent results for the same input")
	}
}

func TestComputeHMACSHA256_DifferentKeys(t *testing.T) {
	data := []byte("test data")
	key1 := "key1"
	key2 := "key2"

	result1 := ComputeHMACSHA256(data, key1)
	result2 := ComputeHMACSHA256(data, key2)

	if result1 == result2 {
		t.Error("ComputeHMACSHA256 should return different results for different keys")
	}
}
