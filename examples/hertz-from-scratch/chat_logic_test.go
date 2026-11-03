package main

import "testing"

func TestReplyToMessage(t *testing.T) {
	for _, tt := range []struct {
		name, message, want, wantError string
	}{
		{"ordinary", "Go", "收到：Go", ""},
		{"inner_space", "你好  Go", "收到：你好  Go", ""},
		{"upper", "/upper Go http", "GO HTTP", ""},
		{"upper_chinese", "/upper  你好 Go  ", "你好 GO", ""},
		{"empty_command", "/upper", "", "text is required after /upper"},
		{"empty_command_text", "/upper   ", "", "text is required after /upper"},
		{"different_prefix", "/uppercase go", "收到：/uppercase go", ""},
		{"case_sensitive", "/UPPER go", "收到：/UPPER go", ""},
		{"tab_not_separator", "/upper\tgo", "收到：/upper\tgo", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			answer, err := replyToMessage(tt.message)
			if answer != tt.want {
				t.Fatalf("answer=%q want=%q", answer, tt.want)
			}
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("unexpected error=%v", err)
				}
			} else if err == nil || err.Error() != tt.wantError {
				t.Fatalf("error=%v want=%q", err, tt.wantError)
			}
		})
	}
}
