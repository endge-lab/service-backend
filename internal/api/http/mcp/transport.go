package mcp

import "encoding/json"

// JSONRPCRequest — сообщение MCP от клиента (JSON-RPC 2.0); описывает тело POST /mcp в OpenAPI.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc" example:"2.0"`
	ID      json.RawMessage `json:"id,omitempty" swaggertype:"integer" example:"1"`
	Method  string          `json:"method" example:"tools/call"`
	Params  json.RawMessage `json:"params,omitempty" swaggertype:"object"`
}

// JSONRPCResponse — ответ MCP-сервера (JSON-RPC 2.0) в формате application/json.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc" example:"2.0"`
	ID      json.RawMessage `json:"id" swaggertype:"integer" example:"1"`
	Result  json.RawMessage `json:"result,omitempty" swaggertype:"object"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError — ошибка протокола JSON-RPC; ошибки инструментов приходят в result с isError.
type JSONRPCError struct {
	Code    int    `json:"code" example:"-32602"`
	Message string `json:"message" example:"unknown tool"`
}
