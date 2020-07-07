package vts

import "fmt"

// InfoPopulator describes something that gathers information about
// a target, for later use in checkers or generators.
type InfoPopulator interface {
	Name() string
	Run(Target, *RunnerEnv, *RuntimeInfo) error
}

// RuntimeInfo tracks information about a target which is generated at runtime,
// and consumed by runners.
type RuntimeInfo struct {
	Data map[InfoPopulator]map[string]interface{}
}

// Get returns the data populated from the populator with the given key.
func (i *RuntimeInfo) Get(ip InfoPopulator, key string) (interface{}, error) {
	if i.Data == nil {
		return nil, fmt.Errorf("runtime info %q.%s is not present", ip.Name(), key)
	}
	if _, ok := i.Data[ip]; !ok {
		return nil, fmt.Errorf("runtime info %q.%s is not present", ip.Name(), key)
	}
	v, ok := i.Data[ip][key]
	if !ok {
		return nil, fmt.Errorf("runtime info %q.%s is not present", ip.Name(), key)
	}
	return v, nil
}

// Set is called by populators to provide information about a target.
func (i *RuntimeInfo) Set(ip InfoPopulator, key string, data interface{}) {
	if i.Data == nil {
		i.Data = make(map[InfoPopulator]map[string]interface{})
	}
	if _, ok := i.Data[ip]; !ok {
		i.Data[ip] = make(map[string]interface{})
	}
	i.Data[ip][key] = data
}

func (i *RuntimeInfo) HasRun(ip InfoPopulator) bool {
	if i.Data == nil {
		return false
	}
	_, run := i.Data[ip]
	return run
}
