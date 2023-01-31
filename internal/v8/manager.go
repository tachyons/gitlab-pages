package v8

import (
	"gitlab.com/gitlab-org/incubation-engineering/jamstack/go-http-v8-adapter/script"
	"io"
)

type ScriptManager struct {
	store map[string]*script.Script
}

const ScriptFilename = "request.js"

func (m *ScriptManager) LoadScript(id string, f io.Reader) (*script.Script, error) {

	s, _ := script.New(ScriptFilename)

	err := s.LoadScriptData(f)

	if err != nil {
		return nil, err
	}

	s.CreateIsolate()
	err = s.Compile()

	m.store[id] = s

	return s, nil
}

func (m *ScriptManager) RetrieveScript(id string) *script.Script {
	s, exists := m.store[id]

	if !exists {
		return nil
	}

	return s
}

func NewManager() ScriptManager {
	return ScriptManager{
		store: make(map[string]*script.Script),
	}
}
