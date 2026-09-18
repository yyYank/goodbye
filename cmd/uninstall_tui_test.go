package cmd

import (
	"bufio"
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/yyYank/goodbye/internal/uninstall"
)

func Testカーソル画面のキー操作と選択(t *testing.T) {
	items := []uninstall.Item{{Name: "foo"}, {Name: "bar"}, {Name: "baz"}}
	for _, tc := range []struct {
		keys string
		want []string
	}{
		{"j \n", []string{"bar"}},
		{"\x1b[B \x1b[B \n", []string{"bar", "baz"}},
		{"ljlkh\n", []string{"bar"}},
		{"kk l\x1b[C\n", []string{"foo"}},
		{"jjjjl\x1b[A\x1b[C\x1b[D\n", []string{"baz"}},
		{"lq", nil}, {"l\x03", nil}, {"\n", nil},
		{"a\n", []string{"foo", "bar", "baz"}}, {"aa\n", nil},
	} {
		t.Run(tc.keys, func(t *testing.T) {
			var out bytes.Buffer
			selected, err := selectCursorItems(items, bufio.NewReader(strings.NewReader(tc.keys)), &out, 2, 70)
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, item := range selected {
				names = append(names, item.Name)
			}
			if !reflect.DeepEqual(names, tc.want) {
				t.Fatalf("got %v want %v", names, tc.want)
			}
			if !strings.Contains(out.String(), "\x1b[?1049l") || !strings.Contains(out.String(), "\x1b[?25h") {
				t.Fatal("画面を復元していない")
			}
		})
	}
}

func Testカーソル画面は途中EOFで削除を確定しない(t *testing.T) {
	var out bytes.Buffer
	_, err := selectCursorItems([]uninstall.Item{{Name: "foo"}}, bufio.NewReader(strings.NewReader("l")), &out, 10, 70)
	if err == nil || !strings.Contains(out.String(), "\x1b[?1049l") {
		t.Fatalf("%v %q", err, out.String())
	}
}
