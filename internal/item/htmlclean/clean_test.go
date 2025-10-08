package htmlclean

import "testing"

func TestCleanHTML(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		max   int
		want  string
	}{
		"strips_tags_and_scripts": {
			input: `<div><p>Hello <strong>world</strong>!<script>bad()</script></p><p>Second&nbsp;line</p></div>`,
			max:   2048,
			want:  "Hello world! Second line",
		},
		"ignores_style_blocks": {
			input: `<style>body{color:red}</style><p>Visible <em>text</em></p>`,
			max:   2048,
			want:  "Visible text",
		},
		"preserves_inline_boundaries_without_spaces": {
			input: `<strong>Hello</strong><em>world</em>`,
			max:   2048,
			want:  "Helloworld",
		},
		"decodes_entities_and_collapses_whitespace": {
			input: "Hello\n\n&amp;nbsp;world\t!",
			max:   2048,
			want:  "Hello world!",
		},
		"handles_parse_errors": {
			input: "Hello &amp; welcome < invalid>",
			max:   2048,
			want:  "Hello & welcome < invalid>",
		},
		"returns_empty_for_blank_input": {
			input: "  \n\t  ",
			max:   2048,
			want:  "",
		},
		"truncates_to_max": {
			input: `<p>` + longString(2100) + `</p>`,
			max:   2000,
			want:  longString(2000),
		},
		"defaults_max_when_non_positive": {
			input: `<div>` + longString(defaultMaxLength+10) + `</div>`,
			max:   0,
			want:  longString(defaultMaxLength),
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := CleanHTML(tc.input, tc.max); got != tc.want {
				t.Fatalf("CleanHTML() = %q, want %q", got, tc.want)
			}
		})
	}
}

func longString(n int) string {
	buf := make([]rune, n)
	for i := range buf {
		buf[i] = 'a'
	}
	return string(buf)
}
