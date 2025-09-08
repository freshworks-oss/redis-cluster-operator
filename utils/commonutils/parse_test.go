package utils

import "testing"

func TestParseRedisMemConf(t *testing.T) {
	type args struct {
		p string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "b",
			args: args{
				p: "12b",
			},
			want:    "12",
			wantErr: false,
		},
		{
			name: "digit",
			args: args{
				p: "1202",
			},
			want:    "1202",
			wantErr: false,
		},
		{
			name: "B",
			args: args{
				p: "12B",
			},
			want:    "12",
			wantErr: false,
		},
		{
			name: "k",
			args: args{
				p: "12k",
			},
			want:    "12000",
			wantErr: false,
		},
		{
			name: "kk",
			args: args{
				p: "12kk",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "kb",
			args: args{
				p: "12kb",
			},
			want:    "12288",
			wantErr: false,
		},
		{
			name: "Kb",
			args: args{
				p: "12Kb",
			},
			want:    "12288",
			wantErr: false,
		},
		{
			name: "m",
			args: args{
				p: "12m",
			},
			want:    "12000000",
			wantErr: false,
		},
		{
			name: "mB",
			args: args{
				p: "12mb",
			},
			want:    "12582912",
			wantErr: false,
		},
		{
			name: "g",
			args: args{
				p: "12g",
			},
			want:    "12000000000",
			wantErr: false,
		},
		{
			name: "gb",
			args: args{
				p: "12gb",
			},
			want:    "12884901888",
			wantErr: false,
		},
		{
			name: "Negative values",
			args: args{
				p: "-100mb",
			},
			want:    "-104857600",
			wantErr: false,
		},
		{
			name: "Very large values (overflow)",
			args: args{
				p: "999999999999gb",
			},
			want:    "3830667723772264448", // This is the actual overflow result
			wantErr: false,
		},
		{
			name: "Mixed case units - Gb",
			args: args{
				p: "100Gb",
			},
			want:    "107374182400",
			wantErr: false,
		},
		{
			name: "Mixed case units - mB",
			args: args{
				p: "50mB",
			},
			want:    "52428800",
			wantErr: false,
		},
		{
			name: "Invalid units",
			args: args{
				p: "100xyz",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Zero value",
			args: args{
				p: "0",
			},
			want:    "0",
			wantErr: false,
		},
		{
			name: "Zero with unit",
			args: args{
				p: "0gb",
			},
			want:    "0",
			wantErr: false,
		},
		{
			name: "Scientific notation",
			args: args{
				p: "1e6",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Decimal values",
			args: args{
				p: "1.5gb",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Only unit without number",
			args: args{
				p: "gb",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Empty string",
			args: args{
				p: "",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Whitespace only",
			args: args{
				p: "   ",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Number with spaces",
			args: args{
				p: " 100 mb ",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Multiple units",
			args: args{
				p: "100mbgb",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Special characters",
			args: args{
				p: "100@mb",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Very large integer",
			args: args{
				p: "9223372036854775807", // Max int64
			},
			want:    "9223372036854775807",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRedisMemConf(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRedisMemConf() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseRedisMemConf() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseRedisMemConfBoundaryValues tests boundary value scenarios
func TestParseRedisMemConfBoundaryValues(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name:        "Boundary - 1 byte",
			input:       "1b",
			expectError: false,
			description: "Should handle minimum meaningful value",
		},
		{
			name:        "Boundary - Max uint32",
			input:       "4294967295",
			expectError: false,
			description: "Should handle max uint32 value",
		},
		{
			name:        "Invalid - Overflow attempt",
			input:       "999999999999999999999999999gb",
			expectError: true,
			description: "Should error on values that would overflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseRedisMemConf(tt.input)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s but got: %v", tt.description, err)
			}
		})
	}
}
