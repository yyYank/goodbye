package transfer

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func testManager(calls *[]string) Manager {
	return Manager{Name: "test", File: "tools.txt", List: func() ([]string, []string, error) {
		return []string{"b@2", "a@1", "a@1"}, []string{"local: unsupported source"}, nil
	},
		InstallArgs: func(s string) ([]string, error) {
			if !strings.Contains(s, "@") {
				return nil, errors.New("invalid")
			}
			return []string{"install", s}, nil
		},
		Run: func(name string, args ...string) ([]byte, error) {
			*calls = append(*calls, args[1])
			if args[1] == "bad@1" {
				return []byte("diagnostic"), errors.New("exit 1")
			}
			return nil, nil
		}}
}

func Testエクスポートは整列重複排除しドライランでは書かない(t *testing.T) {
	var calls []string
	m := testManager(&calls)
	dir := filepath.Join(t.TempDir(), "export")
	var out bytes.Buffer
	if err := m.Export(Options{Dir: dir, DryRun: true, Out: &out}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote directory")
	}
	if !strings.Contains(out.String(), "local: unsupported source") {
		t.Fatal(out.String())
	}
	if err := m.Export(Options{Dir: dir, Out: &out}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, m.File))
	if err != nil || string(data) != "a@1\nb@2\n" {
		t.Fatalf("%q %v", data, err)
	}
	m.List = func() ([]string, []string, error) { return nil, nil, errors.New("unavailable") }
	if err := m.Export(Options{Dir: dir, Out: &out}); err == nil {
		t.Fatal("expected error")
	}
	data, _ = os.ReadFile(filepath.Join(dir, m.File))
	if string(data) != "a@1\nb@2\n" {
		t.Fatal("clobbered export")
	}
}

func Testインポートは全行検証してから実行する(t *testing.T) {
	var calls []string
	m := testManager(&calls)
	dir := t.TempDir()
	var out bytes.Buffer
	for _, input := range []string{"a@1\ninvalid\n", "a@1\n" + strings.Repeat("x", 70000)} {
		if err := os.WriteFile(filepath.Join(dir, m.File), []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
		if err := m.Import(Options{Dir: dir, Out: &out}); err == nil {
			t.Fatal("expected error")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, m.File), []byte("# comment\n\na@1\na@1\nb@2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := m.Import(Options{Dir: dir, DryRun: true, Out: &out}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
	if err := m.Import(Options{Dir: dir, Out: &out}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []string{"a@1", "b@2"}) {
		t.Fatal(calls)
	}
}

func Test継続指定でも失敗を返す(t *testing.T) {
	for _, cont := range []bool{false, true} {
		t.Run(map[bool]string{false: "停止", true: "継続"}[cont], func(t *testing.T) {
			var calls []string
			m := testManager(&calls)
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, m.File), []byte("bad@1\nok@2\n"), 0600); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			err := m.Import(Options{Dir: dir, Continue: cont, Out: &out})
			if err == nil || !strings.Contains(out.String(), "diagnostic") {
				t.Fatalf("%v %s", err, &out)
			}
			want := 1
			if cont {
				want = 2
			}
			if len(calls) != want {
				t.Fatal(calls)
			}
		})
	}
}

func Test空一覧と存在しないファイルを区別する(t *testing.T) {
	var calls []string
	m := testManager(&calls)
	dir := t.TempDir()
	var out bytes.Buffer
	if err := m.Import(Options{Dir: dir, Out: &out}); err == nil {
		t.Fatal("expected missing file error")
	}
	m.List = func() ([]string, []string, error) { return nil, nil, nil }
	if err := m.Export(Options{Dir: dir, Out: &out}); err != nil {
		t.Fatal(err)
	}
	if err := m.Import(Options{Dir: dir, Out: &out}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}
