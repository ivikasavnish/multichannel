package callbacks

import (
	"encoding/json"
	"fmt"
	"multichannel/cmd/typedefs"
)

// CallbackRegistry stores mapping of callback functions
type CallbackRegistry struct {
	callbacks map[string]func(typedefs.Request) interface{}
}

// NewCallbackRegistry creates a new registry
func NewCallbackRegistry() *CallbackRegistry {
	return &CallbackRegistry{
		callbacks: make(map[string]func(typedefs.Request) interface{}),
	}
}

// Execute calls the registered function by name with provided arguments
func (r *CallbackRegistry) Execute(name string, args ...interface{}) ([]byte, error) {
	if name != "REQUEST" || len(args) != 1 {
		return nil, fmt.Errorf("callback %s not found", name)
	}
	byteinput, ok := args[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid input")
	}
	var request typedefs.Request

	err := json.Unmarshal(byteinput, &request)
	if err != nil {
		return nil, err
	}
	//return []byte(fmt.Sprintf("Received request: %v %v", request.Path, request.Method)), nil
	callback, exists := r.callbacks[request.Path]
	if !exists {
		return nil, fmt.Errorf("callback %s not found", name)
	}

	result := callback(request)

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return resultBytes, nil
}

// GetRegisteredCallbacks returns all registered callback names
func (r *CallbackRegistry) GetRegisteredCallbacks() []string {
	var names []string
	for name := range r.callbacks {
		names = append(names, name)
	}
	return names
}

// Register adds a callback function to the registry
func (r *CallbackRegistry) Register(name string, callback func(typedefs.Request) interface{}) error {
	r.callbacks[name] = callback
	return nil
}
