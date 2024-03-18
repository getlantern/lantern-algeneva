package genevahttp

import (
	"bytes"
	"io"
	"net"
	"testing"

	"github.com/getlantern/algeneva"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockReader struct {
	data [][]byte
	idx  int
}

func (r *mockReader) Read(p []byte) (n int, err error) {
	if r.idx >= len(r.data) {
		return 0, io.EOF
	}

	n = copy(p, r.data[r.idx])
	r.idx++
	return n, nil
}

func TestReadAtLeastUntil(t *testing.T) {
	tests := []struct {
		name       string
		readerData [][]byte
		token      []byte
		wantBytes  int
		wantErr    error
	}{
		{
			name:       "token in a single read",
			readerData: [][]byte{[]byte("The hardest battles are fought in mind.")},
			token:      []byte("battles"),
			wantBytes:  39,
		}, {
			name: "token split between two reads",
			readerData: [][]byte{
				[]byte("He's gonna be out in the frickin grapes it's he.. -_-"),
				[]byte("GRAPE..GRAPE..GRAwal"),
				[]byte("doPE..GRAPE.."),
			},
			token:     []byte("waldo"),
			wantBytes: 86,
		}, {
			name:       "empty src",
			readerData: [][]byte{},
			token:      []byte("TOKEN"),
			wantBytes:  0,
			wantErr:    io.EOF,
		}, {
			name:       "EOF before token found",
			readerData: [][]byte{[]byte("Danger Zone! (/.*)/")},
			token:      []byte("TOKEN"),
			wantBytes:  0,
			wantErr:    io.EOF,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dst bytes.Buffer
			src := &mockReader{data: tt.readerData}
			read, err := readAtLeastUntil(src, &dst, tt.token)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantBytes, read)
			assert.Contains(t, dst.String(), string(tt.token))
		})
	}
}

func TestHTTPTransformConnShortWrite(t *testing.T) {
	wrapped, _ := net.Pipe()

	s, err := algeneva.NewHTTPStrategy(algeneva.Strategies["China"][9])
	require.NoError(t, err)

	htc := httpTransformConn{
		Conn:          wrapped,
		httpTransform: s,
	}

	_, err = htc.Write([]byte{'h'})
	require.NoError(t, err)

	_, err = htc.Write([]byte{'i'})
	require.NoError(t, err)
}
