package jobs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jobright/api/internal/models"
)

type SyncResult struct {
	Source   string `json:"source"`
	Ingested int    `json:"ingested"`
	Error    string `json:"error,omitempty"`
}

func (s *Service) SyncSoftwareJobs() ([]SyncResult, error) {
	results := []SyncResult{
		s.syncRemotive(),
		s.syncArbeitnow(),
		s.syncRemoteOK(),
	}
	return results, nil
}

func (s *Service) syncRemotive() SyncResult {
	res := SyncResult{Source: "remotive"}
	req, err := http.NewRequest(http.MethodGet, "https://remotive.com/api/remote-jobs?category=software-dev", nil)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	req.Header.Set("User-Agent", "jobright/1.0")
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		res.Error = err.Error()
		return res
	}
	var payload struct {
		Jobs []struct {
			ID          int    `json:"id"`
			URL         string `json:"url"`
			Title       string `json:"title"`
			CompanyName string `json:"company_name"`
			Description string `json:"description"`
			CandidateRequiredLocation string `json:"candidate_required_location"`
			Salary      string `json:"salary"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		res.Error = err.Error()
		return res
	}
	for _, j := range payload.Jobs {
		if !isSoftwareRelated(j.Title, nil) {
			continue
		}
		job := &models.Job{
			Title:       strings.TrimSpace(j.Title),
			Company:     strings.TrimSpace(j.CompanyName),
			Description: stripHTML(j.Description),
			Location:    nonEmpty(j.CandidateRequiredLocation, "Remote"),
			SourceURL:   strings.TrimSpace(j.URL),
			SalaryRange: strings.TrimSpace(j.Salary),
		}
		if job.Title == "" || job.SourceURL == "" {
			continue
		}
		if err := s.UpsertBySourceURL(job); err == nil {
			res.Ingested++
		}
	}
	return res
}

func (s *Service) syncArbeitnow() SyncResult {
	res := SyncResult{Source: "arbeitnow"}
	req, err := http.NewRequest(http.MethodGet, "https://www.arbeitnow.com/api/job-board-api", nil)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	req.Header.Set("User-Agent", "jobright/1.0")
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		res.Error = err.Error()
		return res
	}
	var payload struct {
		Data []struct {
			Slug        string   `json:"slug"`
			URL         string   `json:"url"`
			Title       string   `json:"title"`
			CompanyName string   `json:"company_name"`
			Description string   `json:"description"`
			Location    string   `json:"location"`
			Tags        []string `json:"tags"`
			Remote      bool     `json:"remote"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		res.Error = err.Error()
		return res
	}
	for _, j := range payload.Data {
		if !isSoftwareRelated(j.Title, j.Tags) {
			continue
		}
		loc := j.Location
		if j.Remote && loc == "" {
			loc = "Remote"
		}
		job := &models.Job{
			Title:       strings.TrimSpace(j.Title),
			Company:     strings.TrimSpace(j.CompanyName),
			Description: stripHTML(j.Description),
			Location:    loc,
			SourceURL:   strings.TrimSpace(j.URL),
		}
		if job.Title == "" || job.SourceURL == "" {
			continue
		}
		if err := s.UpsertBySourceURL(job); err == nil {
			res.Ingested++
		}
	}
	return res
}

func (s *Service) syncRemoteOK() SyncResult {
	res := SyncResult{Source: "remoteok"}
	req, err := http.NewRequest(http.MethodGet, "https://remoteok.com/api?tags=dev", nil)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	req.Header.Set("User-Agent", "jobright/1.0")
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		res.Error = err.Error()
		return res
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		res.Error = err.Error()
		return res
	}
	for _, row := range rows {
		if _, ok := row["id"]; !ok {
			continue
		}
		title, _ := row["position"].(string)
		company, _ := row["company"].(string)
		desc, _ := row["description"].(string)
		loc, _ := row["location"].(string)
		urlStr, _ := row["url"].(string)
		if urlStr == "" {
			if slug, ok := row["slug"].(string); ok && slug != "" {
				urlStr = "https://remoteok.com/remote-jobs/" + slug
			}
		}
		tags := []string{}
		if rawTags, ok := row["tags"].([]any); ok {
			for _, t := range rawTags {
				if s, ok := t.(string); ok {
					tags = append(tags, s)
				}
			}
		}
		if !isSoftwareRelated(title+" "+strings.Join(tags, " "), tags) && !isSoftwareRelated(title, nil) {
			continue
		}
		salary := ""
		if smin, ok := row["salary_min"].(float64); ok && smin > 0 {
			salary = fmt.Sprintf("%.0f+", smin)
		}
		job := &models.Job{
			Title:       strings.TrimSpace(title),
			Company:     strings.TrimSpace(company),
			Description: stripHTML(desc),
			Location:    nonEmpty(loc, "Remote"),
			SourceURL:   strings.TrimSpace(urlStr),
			SalaryRange: salary,
		}
		if job.Title == "" || job.SourceURL == "" {
			continue
		}
		if err := s.UpsertBySourceURL(job); err == nil {
			res.Ingested++
		}
	}
	return res
}

func isSoftwareRelated(title string, tags []string) bool {
	hay := strings.ToLower(title + " " + strings.Join(tags, " "))
	keys := []string{
		"software", "engineer", "developer", "frontend", "backend", "full stack", "fullstack",
		"devops", "sre", "platform", "mobile", "ios", "android", "react", "golang", "python",
		"typescript", "java", "data engineer", "machine learning", "ml engineer", "qa engineer",
	}
	for _, k := range keys {
		if strings.Contains(hay, k) {
			return true
		}
	}
	return false
}

func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	text := strings.Join(strings.Fields(b.String()), " ")
	if len(text) > 8000 {
		return text[:8000]
	}
	return text
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}
