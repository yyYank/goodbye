package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func Testインストールコマンドをmiseに変換する(t *testing.T) {
	for _, tc := range []struct{ input, spec string }{
		{"go install github.com/foo/bar@latest", "go:github.com/foo/bar@latest"},
		{"go install github.com/foo/bar@v1.2.3", "go:github.com/foo/bar@v1.2.3"},
		{"npm install -g prettier", "npm:prettier"},
		{"npm i -g @scope/tool@3.6.2", "npm:@scope/tool@3.6.2"},
		{"pnpm add -g prettier@3.6.2", "npm:prettier@3.6.2"},
		{"cargo install ripgrep", "cargo:ripgrep"},
		{"cargo install ripgrep --version 14.1.1", "cargo:ripgrep@14.1.1"},
		{"uv tool install ruff", "pypi:ruff"},
		{"uv tool install 'ruff==0.9.1'", "pypi:ruff@0.9.1"},
	} {
		for _, stdin := range []bool{false, true} {
			t.Run(tc.input+map[bool]string{true: "標準入力", false: "引数"}[stdin], func(t *testing.T) {
				c := newConvertCommand()
				var out bytes.Buffer
				c.SetOut(&out)
				args := []string{"--to", "mise"}
				if stdin {
					c.SetIn(strings.NewReader(tc.input + "\n"))
				} else {
					args = append(args, tc.input)
				}
				c.SetArgs(args)
				if err := c.Execute(); err != nil {
					t.Fatal(err)
				}
				if got, want := out.String(), "mise use -g "+tc.spec+"\n"; got != want {
					t.Fatalf("got %q, want %q", got, want)
				}
			})
		}
	}
}

func Test曖昧な入力ではコマンドを出力しない(t *testing.T) {
	for _, input := range []string{
		"", "sudo npm install -g foo", "npm install foo", "npm install -g foo bar",
		"npm install -g foo --ignore-scripts", "cargo install foo --git https://example.com/foo.git",
		"cargo install foo --version", "cargo install foo --version ^1.0", "go install example.com/foo",
		"npm install -g foo; touch /tmp/marker", "npm install -g $(touch /tmp/marker)",
		"npm install -g `whoami`", "npm install -g $PACKAGE", "npm install -g foo | sh",
		"npm install -g foo\ncargo install bar", "npm install -g foo && cargo install bar",
		"npm install -g foo > out", "npm install -g './foo'", "npm install -g foo@https://example.com/foo",
		"uv tool install 'ruff>=0.9'", "uv tool install 'ruff[extra]'", "npm install -g 'foo",
		"npm install -g foo@", "go install example.com/foo@", "npm install -g foo\x00",
		"go install example.com/tools/...@latest",
	} {
		t.Run(input, func(t *testing.T) {
			c := newConvertCommand()
			var out, stderr bytes.Buffer
			c.SetOut(&out)
			c.SetErr(&stderr)
			c.SetArgs([]string{"--to", "mise", input})
			if err := c.Execute(); err == nil {
				t.Fatal("エラーにならなかった")
			}
			if out.Len() != 0 {
				t.Fatalf("失敗時の標準出力: %q", out.String())
			}
		})
	}
}

type convertBrokenIO struct{ err error }

func (b convertBrokenIO) Read([]byte) (int, error)  { return 0, b.err }
func (b convertBrokenIO) Write([]byte) (int, error) { return 0, b.err }

func Test変換コマンドの入出力エラーを返す(t *testing.T) {
	failure := errors.New("入出力エラー")
	for _, reading := range []bool{true, false} {
		c := newConvertCommand()
		c.SetErr(&bytes.Buffer{})
		c.SetArgs([]string{"--to", "mise"})
		if reading {
			c.SetIn(convertBrokenIO{failure})
			c.SetOut(&bytes.Buffer{})
		} else {
			c.SetIn(strings.NewReader("npm install -g prettier"))
			c.SetOut(convertBrokenIO{failure})
		}
		if err := c.Execute(); !errors.Is(err, failure) {
			t.Fatalf("got %v", err)
		}
	}
}

func Test変換コマンドが登録されている(t *testing.T) {
	c, _, err := rootCmd.Find([]string{"convert"})
	if err != nil || c.Name() != "convert" || c.Parent() != rootCmd {
		t.Fatalf("変換コマンドが見つからない: %v", err)
	}
}

func Test変換先と引数の数を検証する(t *testing.T) {
	for _, args := range [][]string{{"--to", "", "npm install -g foo"}, {"--to", "brew", "npm install -g foo"}, {"--to", "mise", "one", "two"}} {
		c := newConvertCommand()
		var out bytes.Buffer
		c.SetOut(&out)
		c.SetErr(&bytes.Buffer{})
		c.SetArgs(args)
		if err := c.Execute(); err == nil || out.Len() != 0 {
			t.Fatalf("args=%q err=%v out=%q", args, err, out.String())
		}
	}
}

func Test複数行とexport一覧をまとめて変換する(t *testing.T) {
	for _, tc := range []struct{ from, input, want string }{
		{"", "#!/bin/sh\r\n# tools\r\n\r\ngo install example.com/a@v1.2.3\r\nnpm i -g @scope/tool\ncargo install ripgrep", "mise use -g go:example.com/a@v1.2.3\nmise use -g npm:@scope/tool\nmise use -g cargo:ripgrep\n"},
		{"go", "# exported\nexample.com/a@v1.2.3\nexample.com/b@latest\n", "mise use -g go:example.com/a@v1.2.3\nmise use -g go:example.com/b@latest\n"},
		{"npm", "@scope/tool\nprettier@3.6.2\n", "mise use -g npm:@scope/tool\nmise use -g npm:prettier@3.6.2\n"},
		{"pnpm", "@scope/tool@1.2.3\n", "mise use -g npm:@scope/tool@1.2.3\n"},
		{"cargo", "ripgrep@14.1.1\n", "mise use -g cargo:ripgrep@14.1.1\n"},
		{"uv", "ruff==0.9.1\nblack==1!25.1.0\n", "mise use -g pypi:ruff@0.9.1\nmise use -g 'pypi:black@1!25.1.0'\n"},
	} {
		t.Run(tc.from, func(t *testing.T) {
			c := newConvertCommand()
			var out bytes.Buffer
			c.SetOut(&out)
			c.SetIn(strings.NewReader(tc.input))
			args := []string{"--to", "mise"}
			if tc.from != "" {
				args = append(args, "--from", tc.from)
			}
			c.SetArgs(args)
			if err := c.Execute(); err != nil {
				t.Fatal(err)
			}
			if out.String() != tc.want {
				t.Fatalf("got %q want %q", out.String(), tc.want)
			}
		})
	}
}

func Test不正行を含む一覧は行番号を返し何も出力しない(t *testing.T) {
	for _, tc := range []struct{ from, input, line string }{
		{"", "# comment\nnpm install -g foo\nset -e\n", "line 3"},
		{"", "go install example.com/a@v1.0.0\ninvalid\n", "line 2"},
		{"go", "example.com/a@v1.0.0\ngo install example.com/b@v1.0.0", "line 2"},
		{"npm", "foo\nfoo --ignore-scripts", "line 2"},
		{"cargo", "ripgrep@14.1.1\nfoo@1.0.0 --git evil", "line 2"},
		{"uv", "ruff==0.9.1\nruff>=0.9", "line 2"},
		{"go", "example.com/a@v1.0.0\n$(touch marker)", "line 2"},
		{"unknown", "foo", "unsupported source"},
		{"", "# comment\n\n", "no"},
	} {
		t.Run(tc.from+tc.input, func(t *testing.T) {
			c := newConvertCommand()
			var out bytes.Buffer
			c.SetOut(&out)
			c.SetErr(&bytes.Buffer{})
			c.SetIn(strings.NewReader(tc.input))
			args := []string{"--to", "mise"}
			if tc.from != "" {
				args = append(args, "--from", tc.from)
			}
			c.SetArgs(args)
			err := c.Execute()
			if err == nil || !strings.Contains(err.Error(), tc.line) || out.Len() != 0 {
				t.Fatalf("err=%v out=%q", err, out.String())
			}
		})
	}
}

func Test変換先省略時もmiseに変換する(t *testing.T) {
	for _, tc := range []struct {
		args        []string
		input, want string
	}{
		{[]string{"go install github.com/foo/bar@latest"}, "", "mise use -g go:github.com/foo/bar@latest\n"},
		{nil, "npm install -g prettier\ncargo install ripgrep\n", "mise use -g npm:prettier\nmise use -g cargo:ripgrep\n"},
		{[]string{"--from", "go"}, "example.com/a@v1.2.3\n", "mise use -g go:example.com/a@v1.2.3\n"},
	} {
		c := newConvertCommand()
		var out bytes.Buffer
		c.SetOut(&out)
		c.SetErr(&bytes.Buffer{})
		c.SetIn(strings.NewReader(tc.input))
		c.SetArgs(append([]string{}, tc.args...))
		if err := c.Execute(); err != nil {
			t.Fatal(err)
		}
		if out.String() != tc.want {
			t.Fatalf("got %q want %q", out.String(), tc.want)
		}
	}
}

func Testパッケージ形式から変換元を判別する(t *testing.T) {
	for _, tc := range []struct{ input, from, want string }{
		{"github.com/d-kuro/gwq/cmd/gwq@v0.0.14", "", "go:github.com/d-kuro/gwq/cmd/gwq@v0.0.14"},
		{"golang.org/x/tools/gopls@v0.23.0", "", "go:golang.org/x/tools/gopls@v0.23.0"},
		{"@openai/codex", "", "npm:@openai/codex"},
		{"@scope/tool@1.2.3", "", "npm:@scope/tool@1.2.3"},
		{"ruff==0.9.1", "", "pypi:ruff@0.9.1"},
		{"ripgrep@14.1.1", "cargo", "cargo:ripgrep@14.1.1"},
		{"@scope/tool", "pnpm", "npm:@scope/tool"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			c := newConvertCommand()
			var out bytes.Buffer
			c.SetOut(&out)
			c.SetErr(&bytes.Buffer{})
			args := []string{tc.input}
			if tc.from != "" {
				args = append([]string{"--from", tc.from}, args...)
			}
			c.SetArgs(args)
			if err := c.Execute(); err != nil {
				t.Fatal(err)
			}
			if want := "mise use -g " + tc.want + "\n"; out.String() != want {
				t.Fatalf("got %q want %q", out.String(), want)
			}
		})
	}
}

func Test自動判別は曖昧な指定や明示形式の不一致を拒否する(t *testing.T) {
	for _, args := range [][]string{
		{"prettier"}, {"ripgrep@14.1.1"}, {"owner/repo@v1.0.0"},
		{"https://github.com/owner/repo@v1.0.0"}, {"./local/tool@v1.0.0"},
		{"--from", "cargo", "@scope/tool"}, {"--from", "npm", "github.com/owner/repo@v1.0.0"},
	} {
		c := newConvertCommand()
		var out bytes.Buffer
		c.SetOut(&out)
		c.SetErr(&bytes.Buffer{})
		c.SetArgs(args)
		err := c.Execute()
		if err == nil || out.Len() != 0 {
			t.Fatalf("args=%q err=%v out=%q", args, err, out.String())
		}
		if len(args) == 1 && !strings.Contains(err.Error(), "--from") {
			t.Fatalf("形式指定の案内がない: %v", err)
		}
	}
}

func Testコマンドとパッケージ指定を混在して全行検証する(t *testing.T) {
	input := "github.com/foo/bar@v1.0.0\n@scope/tool\ncargo install ripgrep\n"
	for _, invalid := range []bool{false, true} {
		c := newConvertCommand()
		var out bytes.Buffer
		c.SetOut(&out)
		c.SetErr(&bytes.Buffer{})
		c.SetArgs([]string{})
		data := input
		if invalid {
			data += "ambiguous\n"
		}
		c.SetIn(strings.NewReader(data))
		err := c.Execute()
		if invalid {
			if err == nil || !strings.Contains(err.Error(), "line 4") || out.Len() != 0 {
				t.Fatalf("err=%v out=%q", err, out.String())
			}
		} else if want := "mise use -g go:github.com/foo/bar@v1.0.0\nmise use -g npm:@scope/tool\nmise use -g cargo:ripgrep\n"; err != nil || out.String() != want {
			t.Fatalf("err=%v out=%q", err, out.String())
		}
	}
}
