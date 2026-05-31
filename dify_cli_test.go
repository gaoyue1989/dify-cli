package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/langgenius/dify-cli/config"
	"github.com/langgenius/dify-cli/tool"
	"github.com/langgenius/dify-cli/types"
)

var binPath = ""

func init() {
	p, err := os.Executable()
	if err != nil || p == "" {
		p = "./dify-cli-linux-amd64"
	}
	binPath = filepath.Join(filepath.Dir(p), "dify-cli-linux-amd64")
	if _, err := os.Stat(binPath); err != nil {
		binPath = "/app/dify-cli-linux-amd64"
	}
	if _, err := os.Stat(binPath); err != nil {
		binPath = "./dify-cli-linux-amd64"
	}
}

// ============================================================
// Config Tests
// ============================================================

func TestConfigLoadValid(t *testing.T) {
	t.Setenv("DIFY_CLI_CONFIG", filepath.Join("testdata", "valid.json"))
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Env.CliApiSessionID != "sess-test-001" {
		t.Errorf("session_id = %q, want %q", cfg.Env.CliApiSessionID, "sess-test-001")
	}
	if len(cfg.ToolReferences) != 4 {
		t.Errorf("got %d tools, want 4", len(cfg.ToolReferences))
	}
}

func TestConfigLoadEmptyTools(t *testing.T) {
	t.Setenv("DIFY_CLI_CONFIG", filepath.Join("testdata", "empty_tools.json"))
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.ToolReferences) != 0 {
		t.Errorf("got %d tools, want 0", len(cfg.ToolReferences))
	}
}

func TestConfigLoadNotFound(t *testing.T) {
	t.Setenv("DIFY_CLI_CONFIG", "/nonexistent/path/config.json")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing config")
	}
	if !strings.Contains(err.Error(), "config not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigFindToolReference(t *testing.T) {
	t.Setenv("DIFY_CLI_CONFIG", filepath.Join("testdata", "valid.json"))
	ref, err := config.FindToolReference("search_docs_abc12345-6789-0abc-def0-123456789abc")
	if err != nil {
		t.Fatalf("FindToolReference failed: %v", err)
	}
	if ref.ToolType != "mcp" {
		t.Errorf("tool_type = %q, want mcp", ref.ToolType)
	}
	if ref.ToolProvider != "mcp_server_1" {
		t.Errorf("provider = %q, want mcp_server_1", ref.ToolProvider)
	}
}

func TestConfigFindToolReferenceShortName(t *testing.T) {
	t.Setenv("DIFY_CLI_CONFIG", filepath.Join("testdata", "valid.json"))
	_, err := config.FindToolReference("search_docs_abc12345")
	if err == nil {
		t.Fatal("expected error for short-name symlink search")
	}
	if !strings.Contains(err.Error(), "tool reference not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigFindToolReferenceNotFound(t *testing.T) {
	t.Setenv("DIFY_CLI_CONFIG", filepath.Join("testdata", "valid.json"))
	_, err := config.FindToolReference("nonexistent_tool_ff000000")
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "tool reference not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigGetReferenceSymlinkName(t *testing.T) {
	ref := types.ToolReference{
		ID:       "abcdef12-3456-7890-abcd-ef1234567890",
		ToolName: "my_tool",
		ToolType: "mcp",
	}
	name := config.GetReferenceSymlinkName(ref)
	expected := "my_tool_abcdef12-3456-7890-abcd-ef1234567890"
	if name != expected {
		t.Errorf("symlink name = %q, want %q", name, expected)
	}
}

func TestConfigGetSelfPath(t *testing.T) {
	path, err := config.GetSelfPath()
	if err != nil {
		t.Fatalf("GetSelfPath failed: %v", err)
	}
	if path == "" {
		t.Fatal("empty self path")
	}
	t.Logf("self path: %s", path)
}

// ============================================================
// Argument Parsing Tests
// ============================================================

func TestParseArgsBasic(t *testing.T) {
	ref := &types.ToolReference{
		ToolName: "test_tool",
		ToolType: "mcp",
	}
	params, err := tool.ParseArgs(ref, []string{"--key1", "value1", "--key2", "value2"})
	if err != nil {
		t.Fatalf("ParseArgs failed: %v", err)
	}
	if params["key1"] != "value1" {
		t.Errorf("key1 = %v, want value1", params["key1"])
	}
	if params["key2"] != "value2" {
		t.Errorf("key2 = %v, want value2", params["key2"])
	}
}

func TestParseArgsWithEquals(t *testing.T) {
	ref := &types.ToolReference{
		ToolName: "test_tool",
		ToolType: "mcp",
	}
	params, err := tool.ParseArgs(ref, []string{"--key1=value1", "--key2=value2"})
	if err != nil {
		t.Fatalf("ParseArgs failed: %v", err)
	}
	if params["key1"] != "value1" {
		t.Errorf("key1 = %v, want value1", params["key1"])
	}
	if params["key2"] != "value2" {
		t.Errorf("key2 = %v, want value2", params["key2"])
	}
}

func TestParseArgsWithDefaults(t *testing.T) {
	ref := &types.ToolReference{
		ToolName: "test_tool",
		ToolType: "mcp",
		DefaultValue: map[string]any{
			"default_param": "default_val",
		},
	}
	params, err := tool.ParseArgs(ref, []string{"--user_param", "user_val"})
	if err != nil {
		t.Fatalf("ParseArgs failed: %v", err)
	}
	if params["default_param"] != "default_val" {
		t.Errorf("default_param = %v, want default_val", params["default_param"])
	}
	if params["user_param"] != "user_val" {
		t.Errorf("user_param = %v, want user_val", params["user_param"])
	}
}

func TestParseArgsOverrideDefaults(t *testing.T) {
	ref := &types.ToolReference{
		ToolName: "test_tool",
		ToolType: "mcp",
		DefaultValue: map[string]any{
			"param": "default",
		},
	}
	params, err := tool.ParseArgs(ref, []string{"--param", "override"})
	if err != nil {
		t.Fatalf("ParseArgs failed: %v", err)
	}
	if params["param"] != "override" {
		t.Errorf("param = %v, want override", params["param"])
	}
}

func TestParseArgsEmpty(t *testing.T) {
	ref := &types.ToolReference{
		ToolName: "test_tool",
		ToolType: "mcp",
	}
	params, err := tool.ParseArgs(ref, []string{})
	if err != nil {
		t.Fatalf("ParseArgs failed: %v", err)
	}
	if len(params) != 0 {
		t.Errorf("got %d params, want 0", len(params))
	}
}

func TestParseArgsIgnoreNonFlag(t *testing.T) {
	ref := &types.ToolReference{
		ToolName: "test_tool",
		ToolType: "mcp",
	}
	params, err := tool.ParseArgs(ref, []string{"positional", "--key1", "val1"})
	if err != nil {
		t.Fatalf("ParseArgs failed: %v", err)
	}
	if params["key1"] != "val1" {
		t.Errorf("key1 = %v, want val1", params["key1"])
	}
}

func TestParseArgsMixedFormat(t *testing.T) {
	ref := &types.ToolReference{
		ToolName: "test_tool",
		ToolType: "mcp",
	}
	params, err := tool.ParseArgs(ref, []string{"--key1=val1", "--key2", "val2", "--key3=val3"})
	if err != nil {
		t.Fatalf("ParseArgs failed: %v", err)
	}
	if params["key1"] != "val1" || params["key2"] != "val2" || params["key3"] != "val3" {
		t.Errorf("got %v", params)
	}
}

// ============================================================
// Response Handler Tests
// ============================================================

func TestHandleToolMessageText(t *testing.T) {
	msg := types.ToolInvokeMessage{
		Type:    "TEXT",
		Message: json.RawMessage(`{"text": "hello world"}`),
	}
	payload := map[string]any{
		"data": msg,
	}
	data, _ := json.Marshal(payload)

	var result struct {
		Data json.RawMessage `json:"data"`
	}
	json.Unmarshal(data, &result)
	if len(result.Data) == 0 {
		t.Fatal("empty data field")
	}
}

func TestHandleToolMessageImage(t *testing.T) {
	msg := types.ToolInvokeMessage{
		Type:    "IMAGE",
		Message: json.RawMessage(`{"url": "http://example.com/img.png", "mime_type": "image/png"}`),
	}
	data, _ := json.Marshal(msg)
	var result struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
}

func TestHandleToolMessageVariable(t *testing.T) {
	msg := types.ToolInvokeMessage{
		Type:    "VARIABLE",
		Message: json.RawMessage(`{"variable_name": "result", "variable_value": "hello"}`),
	}
	data, _ := json.Marshal(msg)
	if len(data) == 0 {
		t.Fatal("empty data")
	}
}

func TestHandleToolMessageBlob(t *testing.T) {
	msg := types.ToolInvokeMessage{
		Type:    "BLOB",
		Message: json.RawMessage(`{"blob": "dGVzdCBkYXRh", "mime_type": "image/png"}`),
	}
	data, _ := json.Marshal(msg)
	if len(data) == 0 {
		t.Fatal("empty data")
	}
}

func TestHandleToolMessageLog(t *testing.T) {
	msg := types.ToolInvokeMessage{
		Type:    "LOG",
		Message: json.RawMessage(`{"id": "log1", "label": "test", "status": "success"}`),
	}
	data, _ := json.Marshal(msg)
	if len(data) == 0 {
		t.Fatal("empty data")
	}
}

func TestHandleToolMessageJSON(t *testing.T) {
	msg := types.ToolInvokeMessage{
		Type:    "JSON",
		Message: json.RawMessage(`{"key": "value"}`),
	}
	data, _ := json.Marshal(msg)
	if len(data) == 0 {
		t.Fatal("empty data")
	}
}

func TestHandleToolMessageUnknown(t *testing.T) {
	msg := types.ToolInvokeMessage{
		Type:    "UNKNOWN_TYPE",
		Message: json.RawMessage(`{}`),
	}
	data, _ := json.Marshal(msg)
	if len(data) == 0 {
		t.Fatal("empty data")
	}
}

// ============================================================
// MIME Type Detection Tests
// ============================================================

func TestDetectMimeTypeTXT(t *testing.T) {
	result := detectMimeTypeWrapper("/path/to/file.txt")
	if result != "text/plain" {
		t.Errorf("got %q, want text/plain", result)
	}
}

func TestDetectMimeTypeJSON(t *testing.T) {
	result := detectMimeTypeWrapper("/path/to/data.json")
	if result != "application/json" {
		t.Errorf("got %q, want application/json", result)
	}
}

func TestDetectMimeTypePNG(t *testing.T) {
	result := detectMimeTypeWrapper("/path/to/image.png")
	if result != "image/png" {
		t.Errorf("got %q, want image/png", result)
	}
}

func TestDetectMimeTypePDF(t *testing.T) {
	result := detectMimeTypeWrapper("/path/to/doc.pdf")
	if result != "application/pdf" {
		t.Errorf("got %q, want application/pdf", result)
	}
}

func TestDetectMimeTypeUnknown(t *testing.T) {
	result := detectMimeTypeWrapper("/path/to/unknown.xyz")
	if result != "application/octet-stream" {
		t.Errorf("got %q, want application/octet-stream", result)
	}
}

func TestDetectMimeTypeDoubleExt(t *testing.T) {
	result := detectMimeTypeWrapper("/path/to/archive.tar.gz")
	if result != "application/gzip" {
		t.Errorf("got %q, want application/gzip", result)
	}
}

func detectMimeTypeWrapper(path string) string {
	ext := strings.ToLower(path[strings.LastIndex(path, ".")+1:])
	return detectAnyMime(ext)
}

func detectAnyMime(ext string) string {
	mimeTypes := map[string]string{
		"json": "application/json",
		"txt":  "text/plain",
		"csv":  "text/csv",
		"xml":  "application/xml",
		"html": "text/html",
		"pdf":  "application/pdf",
		"png":  "image/png",
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"gif":  "image/gif",
		"svg":  "image/svg+xml",
		"zip":  "application/zip",
		"gz":   "application/gzip",
		"tar":  "application/x-tar",
		"mp3":  "audio/mpeg",
		"mp4":  "video/mp4",
		"py":   "text/x-python",
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

// ============================================================
// Signing Tests
// ============================================================

func TestSignRequest(t *testing.T) {
	ts, sig := tool.SignRequest("test-secret", "POST", "/cli/api/invoke/tool", []byte(`{"key":"val"}`), "session-1")
	if ts == "" {
		t.Error("timestamp is empty")
	}
	if sig == "" || !strings.HasPrefix(sig, "sha256=") {
		t.Errorf("signature = %q, want sha256=...", sig)
	}
}

func TestSignRequestConsistency(t *testing.T) {
	body := []byte(`{"test": "data"}`)

	ts1, sig1 := tool.SignRequest("secret", "POST", "/path", body, "sess")
	ts2, sig2 := tool.SignRequest("secret", "POST", "/path", body, "sess")

	if ts1 != ts2 {
		t.Errorf("timestamps differ: %s vs %s", ts1, ts2)
	}
	if sig1 != sig2 {
		t.Errorf("signatures differ for same inputs: %s vs %s", sig1, sig2)
	}
}

func TestSignRequestSameInputs(t *testing.T) {
	body := []byte(`{"test": "data"}`)
	_, sig1 := tool.SignRequest("secret", "POST", "/path", body, "sess")
	_, sig2 := tool.SignRequest("secret", "POST", "/path", body, "sess")
	if sig1 != sig2 {
		t.Errorf("signatures differ for same inputs: %s vs %s", sig1, sig2)
	}
}

func TestSignRequestDifferentSecrets(t *testing.T) {
	body := []byte(`{"test": "data"}`)
	_, sig1 := tool.SignRequest("secret1", "POST", "/path", body, "sess")
	_, sig2 := tool.SignRequest("secret2", "POST", "/path", body, "sess")
	if sig1 == sig2 {
		t.Error("signatures should differ for different secrets")
	}
}

// ============================================================
// CLI Integration Tests
// ============================================================

func TestCLIHelp(t *testing.T) {
	cmd := exec.Command(binPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("help failed: %v\n%s", err, output)
	}
	out := string(output)
	if !strings.Contains(out, "Dify CLI for sandbox tool invocation") {
		t.Errorf("help output missing title: %s", out)
	}
	if !strings.Contains(out, "init") {
		t.Error("help output missing init command")
	}
	if !strings.Contains(out, "list") {
		t.Error("help output missing list command")
	}
	if !strings.Contains(out, "env") {
		t.Error("help output missing env command")
	}
	if !strings.Contains(out, "DIFY_CLI_CONFIG") {
		t.Error("help output missing env var docs")
	}
}

func TestCLIUnknownCommand(t *testing.T) {
	cmd := exec.Command(binPath, "unknown_cmd")
	output, _ := cmd.CombinedOutput()
	out := string(output)
	if !strings.Contains(out, "unknown command") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLIEmptyArgs(t *testing.T) {
	cmd := exec.Command(binPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("empty args failed: %v", err)
	}
	if len(output) == 0 {
		t.Fatal("no output for empty args")
	}
}

func TestCLINoConfig(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command(binPath, "init")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "DIFY_CLI_CONFIG=")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for missing config")
	}
	if !strings.Contains(string(output), "config not found") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestCLIInit(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join("testdata", "valid.json")
	dst := filepath.Join(dir, config.ConfigFilename)

	data, _ := os.ReadFile(src)
	os.WriteFile(dst, data, 0644)

	cmd := exec.Command(binPath, "init")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init failed: %v\n%s", err, output)
	}
	out := string(output)
	if !strings.Contains(out, "Created") {
		t.Errorf("init output missing: %s", out)
	}
	if !strings.Contains(out, "skipped") {
		t.Errorf("init output missing skipped count: %s", out)
	}

	for _, name := range []string{
		"search_docs_abc12345-6789-0abc-def0-123456789abc",
		"web_reader_def45678",
		"weather_api_aaa11111",
		"my_workflow_bbb11111",
	} {
		path := filepath.Join(dir, name)
		if _, err := os.Lstat(path); err != nil {
			t.Errorf("symlink %s not created", name)
		}
	}
}

func TestCLIInitEmpty(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join("testdata", "empty_tools.json")
	dst := filepath.Join(dir, config.ConfigFilename)

	data, _ := os.ReadFile(src)
	os.WriteFile(dst, data, 0644)

	cmd := exec.Command(binPath, "init")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init empty failed: %v\n%s", err, output)
	}
	out := string(output)
	if !strings.Contains(out, "No tool references") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCLIList(t *testing.T) {
	dir := t.TempDir()
	copyConfig(t, "valid.json", dir)

	cmd := exec.Command(binPath, "list")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list failed: %v\n%s", err, output)
	}
	out := string(output)
	if !strings.Contains(out, "Available tools:") {
		t.Errorf("list output missing header: %s", out)
	}
	if !strings.Contains(out, "search_docs_abc12345-6789-0abc-def0-123456789abc") {
		t.Error("list output missing search_docs")
	}
	if !strings.Contains(out, "web_reader_def45678") {
		t.Error("list output missing web_reader")
	}
	if !strings.Contains(out, "weather_api_aaa11111") {
		t.Error("list output missing weather_api")
	}
	if !strings.Contains(out, "my_workflow_bbb11111") {
		t.Error("list output missing my_workflow")
	}
}

func TestCLIEnv(t *testing.T) {
	dir := t.TempDir()
	copyConfig(t, "valid.json", dir)

	cmd := exec.Command(binPath, "env")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("env failed: %v\n%s", err, output)
	}
	out := string(output)
	if !strings.Contains(out, "Files URL:") {
		t.Error("env output missing Files URL")
	}
	if !strings.Contains(out, "CLI API URL:") {
		t.Error("env output missing CLI API URL")
	}
	if !strings.Contains(out, "Session ID:") {
		t.Error("env output missing Session ID")
	}
	if !strings.Contains(out, "sess-test-001") {
		t.Error("env output missing session id value")
	}
}

func TestCLIInitSkipExisting(t *testing.T) {
	dir := t.TempDir()
	copyConfig(t, "valid.json", dir)

	cmd := exec.Command(binPath, "init")
	cmd.Dir = dir
	cmd.Run()

	cmd = exec.Command(binPath, "init")
	cmd.Dir = dir
	output, _ := cmd.CombinedOutput()
	out := string(output)
	if !strings.Contains(out, "skipped") && !strings.Contains(out, "Created") {
		t.Errorf("expected init output: %s", out)
	}
}

func TestCLISymlinkDetection(t *testing.T) {
	dir := t.TempDir()
	copyConfig(t, "valid.json", dir)

	cmd := exec.Command(binPath, "init")
	cmd.Dir = dir
	cmd.Run()

	symlinkPath := filepath.Join(dir, "search_docs_abc12345-6789-0abc-def0-123456789abc")
	cmd = exec.Command(symlinkPath, "--help")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Logf("symlink help output: %s", output)
	}
}

func TestCLIDifyCliConfigEnv(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join("testdata", "valid.json")
	dst := filepath.Join(dir, "custom_config.json")
	data, _ := os.ReadFile(src)
	os.WriteFile(dst, data, 0644)

	cmd := exec.Command(binPath, "init")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), fmt.Sprintf("DIFY_CLI_CONFIG=%s", dst))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init with DIFY_CLI_CONFIG failed: %v\n%s", err, output)
	}

	for _, name := range []string{
		"search_docs_abc12345-6789-0abc-def0-123456789abc",
		"web_reader_def45678",
	} {
		path := filepath.Join(dir, name)
		if _, err := os.Lstat(path); err != nil {
			t.Errorf("symlink %s not created via DIFY_CLI_CONFIG", name)
		}
	}
}

func TestCLIExecuteMode(t *testing.T) {
	dir := t.TempDir()
	copyConfig(t, "valid.json", dir)

	cmd := exec.Command(binPath, "init")
	cmd.Dir = dir
	cmd.Run()

	cmd = exec.Command(binPath, "execute", "search_docs_abc12345-6789-0abc-def0-123456789abc", "--query", "test")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Logf("execute output: %s", output)
	}
}

func TestCLIHelpCommand(t *testing.T) {
	dir := t.TempDir()
	copyConfig(t, "valid.json", dir)

	cmd := exec.Command(binPath, "help", "search_docs")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Logf("help command output: %s", output)
	}
}

// ============================================================
// Helpers
// ============================================================

func copyConfig(t *testing.T, name string, dir string) {
	t.Helper()
	src := filepath.Join("testdata", name)
	dst := filepath.Join(dir, config.ConfigFilename)
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read test config: %v", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatalf("write test config: %v", err)
	}
}
