package callbacks

import (
	"fmt"
	"reflect"
)

// CallbackRegistry stores mapping of callback functions
type CallbackRegistry struct {
	callbacks map[string]interface{}
}

// NewCallbackRegistry creates a new registry
func NewCallbackRegistry() *CallbackRegistry {
	return &CallbackRegistry{
		callbacks: make(map[string]interface{}),
	}
}

// Register adds a callback function to the registry
func (r *CallbackRegistry) Register(name string, callback interface{}) error {
	if _, exists := r.callbacks[name]; exists {
		return fmt.Errorf("callback %s already registered", name)
	}
	r.callbacks[name] = callback
	return nil
}

// Execute calls the registered function by name with provided arguments
func (r *CallbackRegistry) Execute(name string, args ...interface{}) ([]interface{}, error) {
	callback, exists := r.callbacks[name]
	if !exists {
		return nil, fmt.Errorf("callback %s not found", name)
	}

	// Get the function's reflect.Value
	fn := reflect.ValueOf(callback)
	if fn.Kind() != reflect.Func {
		return nil, fmt.Errorf("callback %s is not a function", name)
	}

	// Convert args to reflect.Value slice
	var values []reflect.Value
	for _, arg := range args {
		values = append(values, reflect.ValueOf(arg))
	}

	// Call the function
	result := fn.Call(values)

	// Convert results back to interface{}
	var returns []interface{}
	for _, r := range result {
		returns = append(returns, r.Interface())
	}

	return returns, nil
}

// GetRegisteredCallbacks returns all registered callback names
func (r *CallbackRegistry) GetRegisteredCallbacks() []string {
	var names []string
	for name := range r.callbacks {
		names = append(names, name)
	}
	return names
}

