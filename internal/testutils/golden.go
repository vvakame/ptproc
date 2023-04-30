package testutils

import (
	"os"
	"path"

	"github.com/pmezard/go-difflib/difflib"
)

type TestingT interface {
	Helper()
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

func CheckGoldenFile(t TestingT, actual []byte, expectFilePath string) {
	t.Helper()

	expectFileDir := path.Dir(expectFilePath)

	expect, err := os.ReadFile(expectFilePath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(expectFileDir, 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(expectFilePath, actual, 0444)
		if err != nil {
			t.Fatal(err)
		}
		return
	} else if err != nil {
		t.Error(err)
		return
	}

	if string(expect) != string(actual) {
		diff := difflib.UnifiedDiff{
			A:       difflib.SplitLines(string(expect)),
			B:       difflib.SplitLines(string(actual)),
			Context: 5,
		}
		d, err := difflib.GetUnifiedDiffString(diff)
		if err != nil {
			t.Fatal(err)
		}
		t.Error(d)
	}
}
