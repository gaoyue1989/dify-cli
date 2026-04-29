package types

import "encoding/json"

type EnvConfig struct {
	FilesURL        string `json:"files_url" validate:"required"`
	CliApiURL       string `json:"cli_api_url" validate:"required"`
	CliApiSessionID string `json:"cli_api_session_id" validate:"required"`
	CliApiSecret    string `json:"cli_api_secret" validate:"required"`
}

type ToolReference struct {
	ID           string         `json:"id"`
	ToolType     string         `json:"tool_type"`
	ToolName     string         `json:"tool_name"`
	ToolProvider string         `json:"tool_provider"`
	CredentialID *string        `json:"credential_id,omitempty"`
	DefaultValue map[string]any `json:"default_value,omitempty"`
}

type DifyConfig struct {
	Env            EnvConfig       `json:"env"`
	ToolReferences []ToolReference `json:"tool_references"`
}

type ToolDescription struct {
	Human map[string]any `json:"human"`
	LLM   string         `json:"llm"`
}

type ToolIdentity struct {
	Name        string `json:"name"`
	Author      string `json:"author"`
	Label       map[string]any `json:"label"`
	Description map[string]any `json:"description"`
}

type ToolDeclaration struct {
	Identity    ToolIdentity     `json:"identity"`
	Description ToolDescription  `json:"description"`
	Parameters  []ToolParameter  `json:"parameters"`
}

type ToolParameter struct {
	Name     string            `json:"name"`
	Label    map[string]any    `json:"label"`
	Human    map[string]any    `json:"human"`
	LLM      string            `json:"llm"`
	Type     string            `json:"type"`
	Required bool              `json:"required"`
	Default  any               `json:"default,omitempty"`
	Options  []ParameterOption `json:"options,omitempty"`
}

type ParameterOption struct {
	Label map[string]any `json:"label"`
	Value string         `json:"value"`
}

type ToolInvokeMessage struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
	Meta    map[string]any  `json:"meta,omitempty"`
}

type TextMessage struct {
	Text string `json:"text"`
}

type ImageMessage struct {
	URL      string `json:"url"`
	MimeType string `json:"mime_type,omitempty"`
}

type FileMessage struct {
	URL      string `json:"url"`
	Filename string `json:"filename,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

type BlobMessage struct {
	Blob     string `json:"blob"`
	MimeType string `json:"mime_type,omitempty"`
}

type BlobChunkMessage struct {
	ID          string `json:"id"`
	Sequence    int    `json:"sequence"`
	TotalLength int    `json:"total_length"`
	Blob        string `json:"blob"`
	End         bool   `json:"end"`
}

type VariableMessage struct {
	VariableName  string `json:"variable_name"`
	VariableValue any    `json:"variable_value"`
	Stream        bool   `json:"stream,omitempty"`
}

type LogMessage struct {
	ID       string         `json:"id"`
	Label    string         `json:"label"`
	ParentID *string        `json:"parent_id,omitempty"`
	Error    *string        `json:"error,omitempty"`
	Status   string         `json:"status"`
	Data     map[string]any `json:"data"`
	Metadata map[string]any `json:"metadata"`
}

type RetrieverResourceMessage struct {
	RetrieverResources []any  `json:"retriever_resources"`
	Context            string `json:"context"`
}

type FetchToolItem struct {
	ToolType     string  `json:"tool_type"`
	ToolProvider string  `json:"tool_provider"`
	ToolName     string  `json:"tool_name"`
	CredentialID *string `json:"credential_id,omitempty"`
}

type FetchToolBatchResponse struct {
	Tools []ToolDeclaration `json:"tools"`
}

type AppInvokeResponse struct {
	Type string        `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type LLMInvokeChunk struct {
	Type string        `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type InnerAPIResponse struct {
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error"`
}

type SignedURLResponse struct {
	Data struct {
		URL string `json:"url"`
	} `json:"data"`
	Error string `json:"error"`
}

type ToolResponseChunk struct {
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error"`
}
