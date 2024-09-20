package writer_test

import (
	"bytes"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/magodo/terrafix/internal/writer"
	"github.com/stretchr/testify/require"
)

func TestUpdateContent(t *testing.T) {
	cases := []struct {
		name    string
		b       []byte
		updates writer.Updates
		nb      []byte
		hasErr  bool
	}{
		{
			name: "updates have overlap",
			b:    bytes.Repeat([]byte("hello"), 10),
			updates: writer.Updates{
				{
					Range: hcl.Range{
						Start: hcl.Pos{Byte: 0},
						End:   hcl.Pos{Byte: 5},
					},
				},
				{
					Range: hcl.Range{
						Start: hcl.Pos{Byte: 4},
						End:   hcl.Pos{Byte: 6},
					},
				},
			},
			hasErr: true,
		},
		{
			name: "update exceed content length",
			b:    bytes.Repeat([]byte("a"), 10),
			updates: writer.Updates{
				{
					Range: hcl.Range{
						Start: hcl.Pos{Byte: 0},
						End:   hcl.Pos{Byte: 11},
					},
				},
			},
			hasErr: true,
		},
		{
			name: "successful update",
			b:    []byte("01234567890123456789"),
			updates: writer.Updates{
				{
					Range: hcl.Range{
						Start: hcl.Pos{Byte: 0},
						End:   hcl.Pos{Byte: 10},
					},
					Content: []byte("hello"),
				},
				{
					Range: hcl.Range{
						Start: hcl.Pos{Byte: 19},
						End:   hcl.Pos{Byte: 20},
					},
					Content: []byte("world"),
				},
			},
			nb: []byte("hello012345678world"),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			nb, err := writer.UpdateContent(tt.b, tt.updates)
			if tt.hasErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.nb, nb)
		})
	}
}
