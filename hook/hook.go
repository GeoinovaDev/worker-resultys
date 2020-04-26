package hook

import (
	"sync"

	"git.resultys.com.br/motor/service"
)

// Hook ...
type Hook struct {
	mx   *sync.Mutex
	list map[string][]func(*service.Unit)
}

// New ...
func New() *Hook {
	return &Hook{
		list: make(map[string][]func(*service.Unit)),
		mx:   &sync.Mutex{},
	}
}

// On ...
func (h *Hook) On(name string, fn func(*service.Unit)) *Hook {
	h.mx.Lock()
	defer h.mx.Unlock()

	if !h.existName(name) {
		h.list[name] = []func(*service.Unit){}
	}

	h.list[name] = append(h.list[name], fn)

	return h
}

// Off ...
func (h *Hook) Off(name string) *Hook {
	h.mx.Lock()
	defer h.mx.Unlock()

	if h.existName(name) {
		h.list[name] = []func(*service.Unit){}
	}

	return h
}

// Trigger ...
func (h *Hook) Trigger(name string, unit *service.Unit) *Hook {
	h.mx.Lock()
	defer h.mx.Unlock()

	if h.existName(name) {
		for i := 0; i < len(h.list[name]); i++ {
			h.list[name][i](unit)
		}
	}

	return h
}

func (h *Hook) existName(name string) bool {
	if _, ok := h.list[name]; ok {
		return true
	}

	return false
}
