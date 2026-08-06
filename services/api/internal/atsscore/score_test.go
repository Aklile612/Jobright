package atsscore

import "testing"

func TestScoreKeywordCoverage(t *testing.T) {
	resume := "Software engineer. Built REST APIs in Go with Postgres and Redis. Docker, Kubernetes."
	job := "Backend Engineer needing Go, Postgres, Kafka, Kubernetes, Redis, Docker"
	r := Score(resume, "Backend Engineer", job)
	if r.Total < 4 {
		t.Fatalf("expected keywords extracted, got %d: %#v", r.Total, r)
	}
	if r.Covered == 0 {
		t.Fatalf("expected some coverage, present=%v", r.Present)
	}
	if r.MatchScore <= 0 || r.MatchScore > 100 {
		t.Fatalf("bad score %v", r.MatchScore)
	}
	// Kafka should be missing
	foundKafka := false
	for _, m := range append(append([]string{}, r.MissingSkills...), r.MissingKeywords...) {
		if m == "kafka" {
			foundKafka = true
		}
	}
	if !foundKafka {
		t.Fatalf("expected kafka missing, missing=%v %v", r.MissingSkills, r.MissingKeywords)
	}
}
