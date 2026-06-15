package markdown

import "testing"

func TestRenderTable(t *testing.T) {
	t.Parallel()

	html, err := Render("| A | B |\n|---|---|\n| 1 | 2 |")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(html, "<table>") || !contains(html, "<th>") || !contains(html, "<td>") {
		t.Fatalf("expected table html: %s", html)
	}
}

func TestRenderSanitizesScript(t *testing.T) {
	t.Parallel()

	html, err := Render("# Title\n\n<script>alert(1)</script>\n\n**bold**")
	if err != nil {
		t.Fatal(err)
	}
	if contains(html, "<script>") {
		t.Fatalf("script not sanitized: %s", html)
	}
	if !contains(html, "<strong>bold</strong>") && !contains(html, "bold") {
		t.Fatalf("expected rendered markdown: %s", html)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
