package utils

import "testing"

func TestEncodeNix32(t *testing.T) {
	tests := []struct {
		name string
		data [20]byte
		want string
	}{
		{
			name: "zero",
			want: "00000000000000000000000000000000",
		},
		{
			name: "sequential",
			data: [20]byte{
				0x00, 0x01, 0x02, 0x03, 0x04,
				0x05, 0x06, 0x07, 0x08, 0x09,
				0x0a, 0x0b, 0x0c, 0x0d, 0x0e,
				0x0f, 0x10, 0x11, 0x12, 0x13,
			},
			want: "2c91240g1q6hq2qa1440f1h50h1h4080",
		},
		{
			name: "all_ff",
			data: [20]byte{
				0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff,
			},
			want: "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EncodeNix32(tt.data); got != tt.want {
				t.Fatalf("EncodeNix32() = %q, want %q", got, tt.want)
			}
		})
	}
}
