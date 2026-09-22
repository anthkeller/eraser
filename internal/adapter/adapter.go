// Package adapter resolves a broker definition to a removal workflow.
package adapter

import (
	"fmt"
	"strings"

	"github.com/eraser-privacy/eraser/internal/broker"
)

// Action is a normalized, executable removal plan.
type Action struct {
	Method    string
	Adapter   string
	Target    string
	Automated bool
}

// Adapter converts one class of broker definitions into removal actions.
// Site-specific adapters can implement this interface without changing the
// send pipeline.
type Adapter interface {
	Name() string
	Supports(broker.Broker) bool
	Plan(broker.Broker) (Action, error)
}

type Registry struct {
	adapters []Adapter
}

func NewRegistry(adapters ...Adapter) *Registry {
	return &Registry{adapters: adapters}
}

func DefaultRegistry() *Registry {
	return NewRegistry(methodAdapter{broker.RemovalEmail, true}, methodAdapter{broker.RemovalWebForm, true}, methodAdapter{broker.RemovalAPI, true}, methodAdapter{broker.RemovalManual, false})
}

func (r *Registry) Resolve(b broker.Broker) (Adapter, error) {
	requested := strings.ToLower(b.Workflow.Removal.Adapter)
	for _, a := range r.adapters {
		if requested != "" && a.Name() != requested {
			continue
		}
		if a.Supports(b) {
			return a, nil
		}
	}
	if requested != "" {
		return nil, fmt.Errorf("broker %q requests unknown or incompatible adapter %q", b.ID, requested)
	}
	return nil, fmt.Errorf("broker %q has unsupported removal method %q", b.ID, b.RemovalMethod())
}

func (r *Registry) Plan(b broker.Broker) (Action, error) {
	a, err := r.Resolve(b)
	if err != nil {
		return Action{}, err
	}
	return a.Plan(b)
}

type methodAdapter struct {
	method    string
	automated bool
}

func (a methodAdapter) Name() string                  { return a.method }
func (a methodAdapter) Supports(b broker.Broker) bool { return b.RemovalMethod() == a.method }
func (a methodAdapter) Plan(b broker.Broker) (Action, error) {
	target := b.RemovalTarget()
	if a.method != broker.RemovalManual && target == "" {
		return Action{}, fmt.Errorf("broker %q has no target for %s removal", b.ID, a.method)
	}
	return Action{Method: a.method, Adapter: a.Name(), Target: target, Automated: a.automated}, nil
}
