package tool

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"time"

	"github.com/langgenius/dify-cli/types"
)

func SignRequest(secret, method, path string, body []byte, sessionID string) (string, string) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	message := fmt.Sprintf("%s.%s", timestamp, string(body))

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	signature := fmt.Sprintf("sha256=%x", mac.Sum(nil))

	return timestamp, signature
}

func ParseArgs(ref *types.ToolReference, args []string) (map[string]any, error) {
	params := make(map[string]any)

	if ref.DefaultValue != nil {
		for k, v := range ref.DefaultValue {
			params[k] = v
		}
	}

	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "--") {
			continue
		}

		key := args[i][2:]

		if strings.Contains(key, "=") {
			parts := strings.SplitN(key, "=", 2)
			params[parts[0]] = parts[1]
			continue
		}

		i++
		if i >= len(args) {
			return nil, fmt.Errorf("missing value for parameter: %s", key)
		}

		params[key] = args[i]
	}

	return params, nil
}

func FetchToolInfo(ref *types.ToolReference, env *types.EnvConfig) (*types.ToolDeclaration, error) {
	path := "/cli/api/fetch/tools/batch"

	reqBody := map[string]any{
		"tools": []map[string]any{
			{
				"tool_type":      ref.ToolType,
				"tool_provider":  ref.ToolProvider,
				"tool_name":      ref.ToolName,
				"credential_id":  ref.CredentialID,
			},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	timestamp, signature := SignRequest(env.CliApiSecret, "POST", path, bodyBytes, env.CliApiSessionID)

	req, _ := http.NewRequest("POST", env.CliApiURL+path, strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cli-Api-Session-Id", env.CliApiSessionID)
	req.Header.Set("X-Cli-Api-Timestamp", timestamp)
	req.Header.Set("X-Cli-Api-Signature", signature)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data  types.FetchToolBatchResponse `json:"data"`
		Error string                        `json:"error"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal json failed: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("API error: %s", result.Error)
	}

	if len(result.Data.Tools) == 0 {
		return nil, fmt.Errorf("tool not found")
	}

	return &result.Data.Tools[0], nil
}

func PrintHelp(decl *types.ToolDeclaration) {
	fmt.Printf("Tool: %s\n", getLabel(decl.Identity.Label))
	if desc := getLabel(decl.Identity.Description); desc != "" {
		fmt.Printf("Description: %s\n", desc)
	}
	fmt.Println()
	fmt.Println("Parameters:")

	for _, p := range decl.Parameters {
		required := ""
		if p.Required {
			required = " (required)"
		}
		fmt.Printf("  --%s: %s%s\n", p.Name, getLabel(p.Label), required)
		if p.Type != "" {
			fmt.Printf("    Type: %s\n", p.Type)
		}
		if getLabel(p.Human) != "" {
			fmt.Printf("    Description: %s\n", getLabel(p.Human))
		}
	}

	fmt.Println()
	fmt.Printf("Usage: execute %s [--param value ...]\n", decl.Identity.Name)
}

func getLabel(m map[string]any) string {
	if m == nil {
		return ""
	}
	if enUS, ok := m["en_US"]; ok {
		if s, ok := enUS.(string); ok {
			return s
		}
	}
	if zhHans, ok := m["zh_Hans"]; ok {
		if s, ok := zhHans.(string); ok {
			return s
		}
	}
	for _, v := range m {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func Dispatch(ref *types.ToolReference, env *types.EnvConfig, params map[string]any) error {
	for key, value := range params {
		strVal, ok := value.(string)
		if !ok {
			continue
		}
		if !strings.HasPrefix(strVal, "@") {
			continue
		}

		filePath := strings.TrimPrefix(strVal, "@")
		if _, err := os.Stat(filePath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: file not found: %s\n", filePath)
			continue
		}

		uploaded, err := uploadFileToServer(filePath, env)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to upload file: %v\n", err)
			continue
		}
		params[key] = uploaded
	}

	path := "/cli/api/invoke/tool"

	reqBody := map[string]any{
		"tool_type":       ref.ToolType,
		"provider":        ref.ToolProvider,
		"tool":            ref.ToolName,
		"tool_parameters": params,
		"credential_id":   ref.CredentialID,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	timestamp, signature := SignRequest(env.CliApiSecret, "POST", path, bodyBytes, env.CliApiSessionID)

	req, _ := http.NewRequest("POST", env.CliApiURL+path, strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cli-Api-Session-Id", env.CliApiSessionID)
	req.Header.Set("X-Cli-Api-Timestamp", timestamp)
	req.Header.Set("X-Cli-Api-Signature", signature)

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 403 {
			return fmt.Errorf("access denied for tool: %s/%s", ref.ToolProvider, ref.ToolName)
		}
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return processToolResponse(respBody)
}

func processToolResponse(data []byte) error {
	var results []struct {
		Data  json.RawMessage `json:"data"`
		Error string           `json:"error"`
	}

	if err := json.Unmarshal(data, &results); err != nil {
		return handleSingleResponse(data)
	}

	for _, r := range results {
		if r.Error != "" {
			fmt.Fprintf(os.Stderr, "[ERROR] %s\n", r.Error)
			continue
		}
		if err := handleToolMessage(r.Data); err != nil {
			return err
		}
	}

	return nil
}

func handleSingleResponse(data []byte) error {
	var result struct {
		Data  json.RawMessage `json:"data"`
		Error string           `json:"error"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return handleToolMessage(data)
	}
	if result.Error != "" {
		return fmt.Errorf("API error: %s", result.Error)
	}
	return handleToolMessage(result.Data)
}

func handleToolMessage(data json.RawMessage) error {
	var msg struct {
		Type    string           `json:"type"`
		Message json.RawMessage  `json:"message"`
		Meta    map[string]any   `json:"meta"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("unmarshal json failed: %w", err)
	}

	switch msg.Type {
	case "TEXT", "text":
		return handleResponseText(msg.Message)
	case "IMAGE", "image":
		return handleResponseImage(msg.Message)
	case "IMAGE_LINK", "image_link":
		return handleResponseImageLink(msg.Message)
	case "FILE", "file":
		return handleResponseFile(msg.Message)
	case "BLOB", "blob":
		return handleResponseBlob(msg.Message, msg.Meta)
	case "BLOB_CHUNK", "blob_chunk":
		return handleResponseBlobChunk(msg.Message)
	case "JSON", "json":
		return handleResponseJSON(msg.Message)
	case "LINK", "link":
		return handleResponseLink(msg.Message)
	case "LOG", "log":
		return handleResponseLog(msg.Message)
	case "VARIABLE", "variable":
		return handleResponseVariable(msg.Message)
	case "RETRIEVER_RESOURCES", "retriever_resources":
		return handleResponseRetrieverResources(msg.Message)
	case "BINARY_LINK", "binary_link":
		return handleResponseBinaryLink(msg.Message)
	default:
		return nil
	}
}

func handleResponseText(data json.RawMessage) error {
	var m struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	fmt.Print(m.Text)
	return nil
}

func handleResponseImage(data json.RawMessage) error {
	var m struct {
		URL      string `json:"url"`
		MimeType string `json:"mime_type,omitempty"`
	}
	json.Unmarshal(data, &m)
	fmt.Fprintf(os.Stderr, "[image] %s\n", m.URL)
	return nil
}

func handleResponseImageLink(data json.RawMessage) error {
	var m struct {
		URL string `json:"url"`
	}
	json.Unmarshal(data, &m)
	fmt.Fprintf(os.Stderr, "[image] %s\n", m.URL)
	return nil
}

func handleResponseFile(data json.RawMessage) error {
	var m struct {
		URL      string `json:"url"`
		Filename string `json:"filename,omitempty"`
		MimeType string `json:"mime_type,omitempty"`
	}
	json.Unmarshal(data, &m)
	filename := m.Filename
	if filename == "" {
		filename = "file"
	}
	fmt.Fprintf(os.Stderr, "[file] %s", filename)
	if m.URL != "" {
		fmt.Fprintf(os.Stderr, " (%s)", m.URL)
	}
	fmt.Fprintln(os.Stderr)
	return nil
}

func handleResponseBlob(data json.RawMessage, meta map[string]any) error {
	var m struct {
		Blob     string `json:"blob"`
		MimeType string `json:"mime_type,omitempty"`
	}
	json.Unmarshal(data, &m)

	if mime, ok := meta["mime_type"].(string); ok && mime != "" {
		fmt.Fprintf(os.Stderr, "[blob] mime_type=%s\n", mime)
	}
	os.Stdout.Write([]byte(m.Blob))
	return nil
}

func handleResponseBlobChunk(data json.RawMessage) error {
	var m struct {
		ID          string `json:"id"`
		Sequence    int    `json:"sequence"`
		TotalLength int    `json:"total_length"`
		Blob        string `json:"blob"`
		End         bool   `json:"end"`
	}
	json.Unmarshal(data, &m)
	os.Stdout.Write([]byte(m.Blob))
	return nil
}

func handleResponseJSON(data json.RawMessage) error {
	var obj any
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	formatted, _ := json.MarshalIndent(obj, "", "  ")
	fmt.Println(string(formatted))
	return nil
}

func handleResponseLink(data json.RawMessage) error {
	var m struct {
		Text string `json:"text"`
		URL  string `json:"url"`
	}
	json.Unmarshal(data, &m)
	if m.URL != "" {
		fmt.Println(m.URL)
	} else if m.Text != "" {
		fmt.Println(m.Text)
	}
	return nil
}

func handleResponseLog(data json.RawMessage) error {
	var m struct {
		ID       string         `json:"id"`
		Label    string         `json:"label"`
		ParentID *string        `json:"parent_id,omitempty"`
		Error    *string        `json:"error,omitempty"`
		Status   string         `json:"status"`
		Data     map[string]any `json:"data"`
	}
	json.Unmarshal(data, &m)
	fmt.Fprintf(os.Stderr, "[log] id=%s label=%s status=%s\n", m.ID, m.Label, m.Status)
	return nil
}

func handleResponseVariable(data json.RawMessage) error {
	var m struct {
		VariableName  string `json:"variable_name"`
		VariableValue any    `json:"variable_value"`
		Stream        bool   `json:"stream,omitempty"`
	}
	json.Unmarshal(data, &m)

	if m.Stream {
		fmt.Fprintf(os.Stderr, "[variable:stream] %s = %v\n", m.VariableName, m.VariableValue)
	} else {
		fmt.Fprintf(os.Stderr, "[variable] %s = %v\n", m.VariableName, m.VariableValue)
	}
	return nil
}

func handleResponseRetrieverResources(data json.RawMessage) error {
	var m struct {
		RetrieverResources []any  `json:"retriever_resources"`
		Context            string `json:"context"`
	}
	json.Unmarshal(data, &m)
	if m.Context != "" {
		fmt.Fprint(os.Stderr, m.Context)
	}
	return nil
}

func handleResponseBinaryLink(data json.RawMessage) error {
	var m struct {
		URL string `json:"url"`
	}
	json.Unmarshal(data, &m)
	fmt.Fprintf(os.Stderr, "[binary_link] %s\n", m.URL)
	return nil
}

func uploadFileToServer(filePath string, env *types.EnvConfig) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	info, _ := file.Stat()
	filename := info.Name()
	mimeType := detectMimeType(filePath)

	signedPath := "/cli/api/upload/file/request"

	reqBody := map[string]any{
		"filename": filename,
		"mimetype":  mimeType,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	timestamp, signature := SignRequest(env.CliApiSecret, "POST", signedPath, bodyBytes, env.CliApiSessionID)

	req, _ := http.NewRequest("POST", env.CliApiURL+signedPath, strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cli-Api-Session-Id", env.CliApiSessionID)
	req.Header.Set("X-Cli-Api-Timestamp", timestamp)
	req.Header.Set("X-Cli-Api-Signature", signature)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get signed URL: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get signed URL: status %d: %s", resp.StatusCode, string(respBody))
	}

	var urlResp types.SignedURLResponse
	if err := json.Unmarshal(respBody, &urlResp); err != nil {
		return "", fmt.Errorf("unmarshal json failed: %w", err)
	}

	if urlResp.Data.URL == "" {
		return "", fmt.Errorf("response data is nil")
	}

	signedURL := urlResp.Data.URL

	if err := uploadToSignedURL(signedURL, filePath, mimeType); err != nil {
		return "", err
	}

	return signedURL, nil
}

func uploadToSignedURL(signedURL, filePath, mimeType string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create a custom part with the correct MIME type (must match the signature)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "file", info.Name()))
	h.Set("Content-Type", mimeType)
	part, err := writer.CreatePart(h)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequest("POST", signedURL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func detectMimeType(filePath string) string {
	ext := strings.ToLower(filePath[strings.LastIndex(filePath, ".")+1:])

	mimeTypes := map[string]string{
		".json": "application/json",
		".txt":  "text/plain",
		".csv":  "text/csv",
		".xml":  "application/xml",
		".html": "text/html",
		".htm":  "text/html",
		".md":   "text/markdown",
		".yaml": "application/x-yaml",
		".yml":  "application/x-yaml",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".webp": "image/webp",
		".zip":  "application/zip",
		".gz":   "application/gzip",
		".tar":  "application/x-tar",
		".mp3":  "audio/mpeg",
		".mp4":  "video/mp4",
		".wav":  "audio/wav",
		".py":   "text/x-python",
		".js":   "application/x-javascript",
		".ts":   "application/x-typescript",
		".go":   "text/x-go",
		".java": "text/x-java",
	}

	fullExt := strings.ToLower(filePath[strings.LastIndex(filePath, "."):])
	if mime, ok := mimeTypes[fullExt]; ok {
		return mime
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}
