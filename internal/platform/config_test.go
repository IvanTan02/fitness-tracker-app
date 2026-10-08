package platform

import "testing"

func TestLoadConfigAppEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		wantErr     bool
	}{
		{name: "development", environment: "development"},
		{name: "production", environment: "production"},
		{name: "empty", environment: "", wantErr: true},
		{name: "unsupported", environment: "staging", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APP_ENV", tt.environment)
			t.Setenv("DATABASE_URL", "postgresql://example")
			t.Setenv("SUPABASE_URL", "https://example.supabase.co")
			t.Setenv("SUPABASE_PUBLISHABLE_KEY", "sb_publishable_example")

			cfg, err := LoadConfig()
			if tt.wantErr {
				if err == nil {
					t.Fatal("LoadConfig() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}
			if cfg.AppEnvironment != tt.environment {
				t.Fatalf("AppEnvironment = %q, want %q", cfg.AppEnvironment, tt.environment)
			}
		})
	}
}
