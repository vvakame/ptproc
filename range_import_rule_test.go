package ptproc

import (
	"bytes"
	"context"
	"io"
	"os"
	"reflect"
	"regexp"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
)

func Test_rangeImportRule_Apply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		startRegExp   *regexp.Regexp
		endRegExp     *regexp.Regexp
		inputFileName string
		rangeName     string
		input         string
		output        string
		wantErr       bool
	}{
		{
			name:          "basic",
			inputFileName: "test.txt",
			rangeName:     "name1",
			input: heredoc.Doc(`
				range:name1
				a
				range.end
			`),
			output: heredoc.Doc(`
				a
			`),
			wantErr: false,
		},
		{
			name:          "multiple",
			inputFileName: "test.txt",
			rangeName:     "name2",
			input: heredoc.Doc(`
				range:name1
				a
				range.end
				range:name2
				b
				range.end
			`),
			output: heredoc.Doc(`
				b
			`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			rule, err := NewRangeImportRule(&RangeImportRuleConfig{
				Name:        tt.rangeName,
				StartRegExp: tt.startRegExp,
				EndRegExp:   tt.endRegExp,
			})
			if err != nil {
				t.Fatal(err)
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
