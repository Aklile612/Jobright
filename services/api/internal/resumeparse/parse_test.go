package resumeparse

import "testing"

func TestExtractPDFParenURIs(t *testing.T) {
	pdf := []byte(`obj << /Type /Annot /A << /S /URI /URI (https://linkedin.com/in/testuser) >> >>
/URI (https://github.com/testuser)
/URI (https://example.com/path\(x\))
`)
	links := extractPDFParenURIs(pdf)
	joined := ""
	for _, l := range links {
		joined += l + "\n"
	}
	for _, want := range []string{
		"https://linkedin.com/in/testuser",
		"https://github.com/testuser",
		"https://example.com/path(x)",
	} {
		if !contains(links, want) {
			t.Fatalf("missing %s in %v", want, links)
		}
	}
}

func TestParseProfileSkillsAndLinks(t *testing.T) {
	text := `Aklile Mente
Software Engineer
Addis Ababa, Ethiopia

Skills: Go, React, TypeScript, PostgreSQL, Docker

Education
BSc Computer Science — Addis Ababa University

Experience
Built APIs and dashboards for hiring workflows.
`
	draft := ParseProfile(text, []string{"https://linkedin.com/in/aklile", "https://github.com/Aklile612"})
	if draft.LinkedIn == "" || draft.GitHub == "" {
		t.Fatalf("links not parsed: %+v", draft)
	}
	if len(draft.Skills) < 3 {
		t.Fatalf("skills not stored: %v", draft.Skills)
	}
	if draft.Education == "" {
		t.Fatalf("education missing")
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
