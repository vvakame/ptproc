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

func Test_mapfileRule_Apply(t *testing.T) {
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
					return "external.txt content"
				default:
					t.Fatalf("unexpected external file: %s", filePath)
					return ""
				}
			},
			inputFileName: "test.txt",
			input: heredoc.Doc(`
				mapfile:external.txt
				mapfile.end
			`),
			output: heredoc.Doc(`
				mapfile:external.txt
				external.txt content
				mapfile.end
			`),
			wantErr: false,
		},
		{
			name: "already have content",
			externalFile: func(t *testing.T, filePath string) string {
				switch filePath {
				case "external.txt":
					return "external.txt content"
				default:
					t.Fatalf("unexpected external file: %s", filePath)
					return ""
				}
			},
			inputFileName: "test.txt",
			input: heredoc.Doc(`
				mapfile:external.txt
				old content
				mapfile.end
			`),
			output: heredoc.Doc(`
				mapfile:external.txt
				external.txt content
				mapfile.end
			`),
			wantErr: false,
		},
		{
			name:          "no replace",
			inputFileName: "test.txt",
			input: heredoc.Doc(`
				foobar!
			`),
			output: heredoc.Doc(`
				foobar!
			`),
			wantErr: false,
		},
		{
			name: "no end directive",
			externalFile: func(t *testing.T, filePath string) string {
				switch filePath {
				case "external.txt":
					return "external.txt content"
				default:
					t.Fatalf("unexpected external file: %s", filePath)
					return ""
				}
			},
			inputFileName: "test.txt",
			input: heredoc.Doc(`
				mapfile:external.txt
			`),
			wantErr: true,
		},
		{
			name: "html comment like",
			externalFile: func(t *testing.T, filePath string) string {
				switch filePath {
				case "external.txt":
					return "external.txt content"
				default:
					t.Fatalf("unexpected external file: %s", filePath)
					return ""
				}
			},
			inputFileName: "test.txt",
			input: heredoc.Doc(`
				<!-- mapfile:external.txt -->
				<!-- mapfile.end -->
			`),
			output: heredoc.Doc(`
				<!-- mapfile:external.txt -->
				external.txt content
				<!-- mapfile.end -->
			`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			rule := &mapfileRule{
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
