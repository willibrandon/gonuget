package main

import (
	"encoding/json"
	"fmt"
)

// Placeholder handlers for non-signature operations
// Signature handlers are implemented in handlers_signature.go

// Version handlers
type CompareVersionsHandler struct{}

func (h *CompareVersionsHandler) ErrorCode() string { return "VER_001" }
func (h *CompareVersionsHandler) Handle(data json.RawMessage) (interface{}, error) {
	return nil, fmt.Errorf("not yet implemented")
}

type ParseVersionHandler struct{}

func (h *ParseVersionHandler) ErrorCode() string { return "VER_002" }
func (h *ParseVersionHandler) Handle(data json.RawMessage) (interface{}, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// Framework handlers
type CheckFrameworkCompatHandler struct{}

func (h *CheckFrameworkCompatHandler) ErrorCode() string { return "FW_001" }
func (h *CheckFrameworkCompatHandler) Handle(data json.RawMessage) (interface{}, error) {
	return nil, fmt.Errorf("not yet implemented")
}

type ParseFrameworkHandler struct{}

func (h *ParseFrameworkHandler) ErrorCode() string { return "FW_002" }
func (h *ParseFrameworkHandler) Handle(data json.RawMessage) (interface{}, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// Package handlers
type ReadPackageHandler struct{}

func (h *ReadPackageHandler) ErrorCode() string { return "PKG_001" }
func (h *ReadPackageHandler) Handle(data json.RawMessage) (interface{}, error) {
	return nil, fmt.Errorf("not yet implemented")
}

type BuildPackageHandler struct{}

func (h *BuildPackageHandler) ErrorCode() string { return "PKG_002" }
func (h *BuildPackageHandler) Handle(data json.RawMessage) (interface{}, error) {
	return nil, fmt.Errorf("not yet implemented")
}
