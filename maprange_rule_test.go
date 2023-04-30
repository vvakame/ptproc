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

func Test_maprangeRule_Apply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		startMarket   *regexp.Regexp
		endMarket     *regexp.Regexp
		externalFile  func(t *testing.T, filePath string) string
		inputFileName string
		input         string
		output        string
		wantErr       bool
	}{
		{
			name: "basic",
			externalFile: func(t *testing.T, filePath string) string {
				switch filePath {
				case "external.txt":
					return heredoc.Doc(`
						test1
						range:name
						test2
						range.end
						test3
					`)
				default:
					t.Fatalf("unexpected external file: %s", filePath)
					return ""
				}
			},
			inputFileName: "test.txt",
			input: heredoc.Doc(`
				maprange:external.txt,name
				maprange.end
			`),
			output: heredoc.Doc(`
				maprange:external.txt,name
				test2
				maprange.end
			`),
			wantErr: false,
		},
		{
			name: "multiple range",
			externalFile: func(t *testing.T, filePath string) string {
				switch filePath {
				case "external.txt":
					return heredoc.Doc(`
						range:name1
						name1
						range.end
						range:name2
						name2
						range.end
					`)
				default:
					t.Fatalf("unexpected external file: %s", filePath)
					return ""
				}
			},
			inputFileName: "test.txt",
			input: heredoc.Doc(`
				maprange:external.txt,name2
				maprange.end
			`),
			output: heredoc.Doc(`
				maprange:external.txt,name2
				name2
				maprange.end
			`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			rule := &maprangeRule{
				startRegExp: tt.startMarket,
				endRegExp:   tt.endMarket,
			}

			proc, err := NewProcessor(&ProcessorConfig{
				OpenFile: func(filePath string) (io.Reader, error) {
					if tt.inputFileName == filePath {
						return bytes.NewBufferString(tt.input), nil
					}
					if tt.externalFile == nil {
						return nil, os.ErrNotExist
					}
					s := tt.externalFile(t, filePath)
					return bytes.NewBufferString(s), nil
				},
				Rules: []Rule{rule},
			})
			if err != nil {
				t.Fatal(err)
			}

			output, err := proc.ProcessFile(ctx, "test.txt")
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
