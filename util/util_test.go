package util

import "testing"

func TestGetMapKeysAsString(t *testing.T) {
	type args struct {
		input map[string]string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"empty string for nil", args{input: nil}, ""},
		{"empty string for empty map", args{input: map[string]string{}}, ""},
		{"single key", args{input: map[string]string{"key": "value"}}, "key"},
		{"many keys", args{input: map[string]string{"key1": "value1", "key2": "value2"}}, "key1, key2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMapKeysAsString(tt.args.input); got != tt.want {
				t.Errorf("GetMapKeysAsString() = %v, want %v", got, tt.want)
			}
		})
	}
}
