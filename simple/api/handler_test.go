package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fabienChaillou/terraform-cmd/api"
	"github.com/fabienChaillou/terraform-cmd/internal/terraform"
)

// newTestServer builds a dry-run server (no Temporal executor).
func newTestServer() *httptest.Server {
	registry := terraform.NewRegistry()
	dispatcher := terraform.NewDispatcher(registry)
	router := api.NewRouterDryRun(dispatcher)
	return httptest.NewServer(router)
}

func postJSON(t *testing.T, server *httptest.Server, payload any) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(
		server.URL+"/terraform/command",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

// ─── 200 success cases ────────────────────────────────────────────────────────

func TestAPI_Huma_ValidWorkspaceList(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	resp := postJSON(t, srv, map[string]any{
		"action":      "workspace",
		"sub_command": "list",
	})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	result := body["result"].(map[string]any)
	if result["command"] != "workspace" {
		t.Errorf("expected command=workspace, got %v", result["command"])
	}
	if result["valid"] != true {
		t.Error("expected result.valid=true")
	}
}

func TestAPI_Huma_WorkspaceNew_Valid(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	resp := postJSON(t, srv, map[string]any{
		"action":      "workspace",
		"sub_command": "new",
		"name":        "staging",
	})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	result := body["result"].(map[string]any)
	if result["sub_command"] != "new" {
		t.Errorf("expected sub_command=new, got %v", result["sub_command"])
	}
}

func TestAPI_Huma_PlanCleanPayload(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	resp := postJSON(t, srv, map[string]any{
		"action": "plan",
		"out":    "tfplan",
		"target": []string{"aws_instance.web"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	result := body["result"].(map[string]any)
	payload := result["payload"].(map[string]any)
	if payload["out"] != "tfplan" {
		t.Errorf("expected out=tfplan in cleaned payload, got %v", payload["out"])
	}
}

func TestAPI_Huma_Init_Minimal(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	resp := postJSON(t, srv, map[string]any{"action": "init"})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAPI_Huma_Apply_WithAutoApprove(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	resp := postJSON(t, srv, map[string]any{
		"action":       "apply",
		"auto_approve": true,
		"parallelism":  5,
	})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	result := body["result"].(map[string]any)
	p := result["payload"].(map[string]any)
	if p["parallelism"] == nil {
		t.Error("expected parallelism in payload")
	}
}

func TestAPI_Huma_Destroy_Valid(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	resp := postJSON(t, srv, map[string]any{
		"action":       "destroy",
		"auto_approve": true,
	})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAPI_Huma_Import_Valid(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	resp := postJSON(t, srv, map[string]any{
		"action":  "import",
		"address": "aws_instance.web",
		"id":      "i-1234567890abcdef0",
	})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// ─── 422 validation error cases ───────────────────────────────────────────────

func TestAPI_Huma_MissingAction(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	// Huma rejects missing required field at schema level → 422
	resp := postJSON(t, srv, map[string]any{"out": "tfplan"})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

func TestAPI_Huma_InvalidAction(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	resp := postJSON(t, srv, map[string]any{"action": "nuke"})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

func TestAPI_Huma_DestroyWithoutAutoApprove(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	resp := postJSON(t, srv, map[string]any{"action": "destroy"})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
	// Huma wraps errors as RFC 9457 Problem Details: { "title": …, "detail": … }
	body := decodeBody(t, resp)
	detail, _ := body["detail"].(string)
	if detail == "" {
		t.Error("expected non-empty detail in Problem Details response")
	}
}

func TestAPI_Huma_WorkspaceMissingName(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	resp := postJSON(t, srv, map[string]any{
		"action":      "workspace",
		"sub_command": "new",
		// name intentionally missing
	})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

func TestAPI_Huma_StateMv_MissingDestination(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	resp := postJSON(t, srv, map[string]any{
		"action":      "state",
		"sub_command": "mv",
		"address":     "aws_instance.old",
		// destination intentionally missing
	})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

func TestAPI_Huma_ImportMissingFields(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	resp := postJSON(t, srv, map[string]any{"action": "import"})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

// ─── 400 bad JSON ─────────────────────────────────────────────────────────────

func TestAPI_Huma_InvalidJSON(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	resp, _ := http.Post(
		srv.URL+"/terraform/command",
		"application/json",
		bytes.NewBufferString("not-json{{"),
	)
	// Huma returns 400 for malformed JSON
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// ─── OpenAPI spec endpoint ────────────────────────────────────────────────────

func TestAPI_Huma_OpenAPISpec(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 on /openapi.json, got %d", resp.StatusCode)
	}
	spec := decodeBody(t, resp)
	if spec["openapi"] == nil {
		t.Error("expected openapi key in spec")
	}
	paths, _ := spec["paths"].(map[string]any)
	if paths["/terraform/command"] == nil {
		t.Error("expected /terraform/command in OpenAPI paths")
	}
}
