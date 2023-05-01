package ptproc

import (
	"bytes"
	"context"
	"io"
	"os"
	"reflect"
	"testing"
)

func Test_reindentRule_Apply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		indentLevel   int
		inputFileName string
		input         string
		output        string
		wantErr       bool
	}{
		{
			name:          "basic tab",
			inputFileName: "test.txt",
			input:         "\tline1\n\tline2\n",
			output:        "  line1\n  line2\n",
			wantErr:       false,
		},
		{
			name:          "4 to 2, 8 to 4",
			inputFileName: "test.txt",
			input:         "    line1\n        line2\n",
			output:        "  line1\n    line2\n",
			wantErr:       false,
		},
		{
			name:          "4 to 2, 2 same lines",
			inputFileName: "test.txt",
			input:         "    line1\n    line2\n",
			output:        "  line1\n  line2\n",
			wantErr:       false,
		},
		{
			name:          "no indent",
			inputFileName: "test.txt",
			input:         "line1\nline2\n",
			output:        "line1\nline2\n",
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			rule := &reindentRule{
				indentLevel: tt.indentLevel,
			}

			proc, err := NewProcessor(&ProcessorConfig{
				OpenFile: func(filePath string) (io.Reader, error) {
					if tt.inputFileName == filePath {
						return bytes.NewBufferString(tt.input), nil
					}
					return nil, os.ErrNotExist
				},
				Rules: []Rule{rule},
			})
			if err != nil {
				t.Fatal(err)
			}

			output, err := proc.ProcessFile(ctx, tt.inputFileName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			} else {
				t.Logf("err = %v", err)
			}

			if !reflect.DeepEqual(output, tt.output) {
				t.Errorf("got = %v, want %v", output, tt.output)
			}
		})
	}
}
