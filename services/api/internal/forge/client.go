package forge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

type AuthResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    *int   `json:"expires_in"`
	TokenType    string `json:"token_type"`
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
}

type ResumeUploadResult struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	FileURL string `json:"file_url"`
}

type JobSummary struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Company string `json:"company"`
}

type ATSReport struct {
	ID              string   `json:"id"`
	ResumeID        string   `json:"resume_id"`
	JobID           string   `json:"job_id"`
	ResumeVersionID *string  `json:"resume_version_id"`
	AnalysisStage   string   `json:"analysis_stage"`
	MatchScore      float64  `json:"match_score"`
	MissingKeywords []string `json:"missing_keywords"`
	Suggestions     []string `json:"suggestions"`
	Strengths       []string `json:"strengths"`
	Weaknesses      []string `json:"weaknesses"`
}

type ResumeVersion struct {
	ID             string          `json:"id"`
	ResumeID       string          `json:"resume_id"`
	VersionNumber  int             `json:"version_number"`
	SourceJobID    *string         `json:"source_job_id"`
	OptimizedJSON  json.RawMessage `json:"optimized_json"`
	DiffJSON       json.RawMessage `json:"diff_json"`
}

type OptimizeResult struct {
	Version    ResumeVersion `json:"version"`
	InitialATS ATSReport     `json:"initial_ats"`
	FinalATS   ATSReport     `json:"final_ats"`
}

type apiError struct {
	Detail string `json:"detail"`
}

func (c *Client) Register(email, password, name string) (*AuthResult, error) {
	return c.auth("/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": password,
		"name":     name,
	})
}

func (c *Client) Login(email, password string) (*AuthResult, error) {
	return c.auth("/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	})
}

func (c *Client) Refresh(refreshToken string) (*AuthResult, error) {
	return c.auth("/api/v1/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	})
}

func (c *Client) auth(path string, body any) (*AuthResult, error) {
	var out AuthResult
	if err := c.doJSON(http.MethodPost, path, "", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UploadResume(token, name, filename string, content []byte, contentType string) (*ResumeUploadResult, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("name", name)
	part, err := w.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(content); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/resumes", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", w.FormDataContentType())

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, decodeAPIError(res.StatusCode, body)
	}
	var out ResumeUploadResult
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ParseResume(token, resumeID string) error {
	return c.doJSON(http.MethodPost, "/api/v1/resumes/"+resumeID+"/parse", token, nil, &map[string]any{})
}

func (c *Client) CreateJob(token, title, company, rawText, url string) (*JobSummary, error) {
	payload := map[string]any{
		"title":    title,
		"company":  company,
		"raw_text": rawText,
	}
	if url != "" {
		payload["url"] = url
	}
	var out JobSummary
	if err := c.doJSON(http.MethodPost, "/api/v1/jobs", token, payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ParseJob(token, jobID string) error {
	return c.doJSON(http.MethodPost, "/api/v1/jobs/"+jobID+"/parse", token, nil, &map[string]any{})
}

func (c *Client) Analyze(token, resumeID, jobID string) (*ATSReport, error) {
	var out ATSReport
	if err := c.doJSON(http.MethodPost, "/api/v1/ats/analyze", token, map[string]string{
		"resume_id": resumeID,
		"job_id":    jobID,
	}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Optimize(token, resumeID, jobID string) (*OptimizeResult, error) {
	var out OptimizeResult
	if err := c.doJSON(http.MethodPost, "/api/v1/resumes/"+resumeID+"/optimize", token, map[string]string{
		"job_id": jobID,
	}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ExportResume(token, resumeID, format string, version *int) ([]byte, string, error) {
	path := fmt.Sprintf("/api/v1/resumes/%s/export?format=%s", resumeID, format)
	if version != nil {
		path = fmt.Sprintf("%s&version=%d", path, *version)
	}
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}
	if res.StatusCode >= 300 {
		return nil, "", decodeAPIError(res.StatusCode, body)
	}
	return body, res.Header.Get("Content-Type"), nil
}

func (c *Client) doJSON(method, path, token string, payload any, out any) error {
	var reader io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode >= 300 {
		return decodeAPIError(res.StatusCode, body)
	}
	if out == nil || res.StatusCode == http.StatusNoContent || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

func decodeAPIError(status int, body []byte) error {
	var ae apiError
	if err := json.Unmarshal(body, &ae); err == nil && ae.Detail != "" {
		return fmt.Errorf("resume_forge %d: %s", status, ae.Detail)
	}
	return fmt.Errorf("resume_forge %d: %s", status, string(body))
}
