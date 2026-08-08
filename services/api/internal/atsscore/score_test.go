package atsscore

import "testing"

func TestScoreKeywordCoverage(t *testing.T) {
	resume := "Software engineer. Built REST APIs in Go with Postgres and Redis. Docker, Kubernetes."
	job := "Backend Engineer needing Go, Postgres, Kafka, Kubernetes, Redis, Docker, observability"
	r := Score(resume, "Backend Engineer", job)
	if r.Total < 4 {
		t.Fatalf("expected keywords extracted, got %d present=%v missing=%v", r.Total, r.Present, r.MissingKeywords)
	}
	if r.Covered == 0 {
		t.Fatalf("expected some coverage, present=%v", r.Present)
	}
	foundKafka := false
	for _, m := range r.MissingKeywords {
		if m == "kafka" {
			foundKafka = true
		}
	}
	if !foundKafka {
		t.Fatalf("expected kafka in MissingKeywords, got %v (skills=%v)", r.MissingKeywords, r.MissingSkills)
	}
	if len(r.MissingKeywords) == 0 {
		t.Fatal("MissingKeywords should not be empty when gaps exist")
	}
}

func TestIgnoresFluffWords(t *testing.T) {
	resume := "Go Postgres Redis Docker"
	job := "Need excellent communication, collaboration, documentation, and Go with Kafka"
	r := Score(resume, "Engineer", job)
	for _, p := range r.Present {
		if p == "communication" || p == "documentation" || p == "collaborate" {
			t.Fatalf("fluff should not be scored: %v", r.Present)
		}
	}
	for _, m := range r.MissingKeywords {
		if m == "communication" || m == "documentation" {
			t.Fatalf("fluff should not appear as missing: %v", r.MissingKeywords)
		}
	}
	found := false
	for _, m := range r.MissingKeywords {
		if m == "kafka" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected kafka missing, got %#v score=%v", r, r.MatchScore)
	}
}
