package scm

type Registry struct {
	adapters []Adapter
	byType   map[Type]Adapter
}

func NewRegistry(adapters ...Adapter) *Registry {
	registry := &Registry{
		adapters: make([]Adapter, 0, len(adapters)),
		byType:   make(map[Type]Adapter, len(adapters)),
	}

	for _, adapter := range adapters {
		if adapter == nil {
			continue
		}

		registry.adapters = append(registry.adapters, adapter)
		registry.byType[adapter.Type()] = adapter
	}

	return registry
}

func (r *Registry) All() []Adapter {
	adapters := make([]Adapter, len(r.adapters))
	copy(adapters, r.adapters)
	return adapters
}

func (r *Registry) For(repoType Type) (Adapter, bool) {
	adapter, ok := r.byType[repoType]
	return adapter, ok
}
