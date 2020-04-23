package worker

import (
	"fmt"
	"sync"
	"time"

	"git.resultys.com.br/lib/lower/promise"
	"git.resultys.com.br/motor/models/token"
	"git.resultys.com.br/motor/service"
)

// Worker struct
type Worker struct {
	services []service.Service
	running  map[string]*service.Unit

	mutex *sync.Mutex
}

// New ...
func New() *Worker {
	w := &Worker{}

	w.mutex = &sync.Mutex{}
	w.services = []service.Service{}

	w.running = make(map[string]*service.Unit)

	return w
}

// Pipe ...
func (w *Worker) Pipe(service service.Service) *Worker {
	w.services = append(w.services, service)

	return w
}

// Wait ...
func (w *Worker) Wait() *Worker {
	w.services = append(w.services, nil)

	return w
}

// Run ...
func (w *Worker) Run(unit *service.Unit) *promise.Promise {
	w.lock()
	if w.existUnit(unit) {
		w.unlock()
		return w.getUnit(unit.Token).Promise
	}

	w.addUnit(unit)
	w.unlock()

	w.runServices(0, unit)

	return unit.Promise
}

// Exist ...
func (w *Worker) Exist(unit *service.Unit) bool {
	w.lock()
	u := w.getUnit(unit.Token)
	w.unlock()

	return u != nil
}

// Load ...
func (w *Worker) Load() *Worker {
	for i := 0; i < len(w.services); i++ {
		if w.services[i] == nil {
			continue
		}

		w.services[i].Load()
	}

	return w
}

// Reload ...
func (w *Worker) Reload() *Worker {
	for i := 0; i < len(w.services); i++ {
		if w.services[i] == nil {
			continue
		}

		w.services[i].Reload()
	}

	return w
}

// Stats ...
func (w *Worker) Stats() {
	var elapsed time.Duration

	for _, service := range w.services {
		elapsed += service.Stats()
	}

	fmt.Println("--------------------------------")
	fmt.Println("TEMPO TOTAL ROUTINES = ", elapsed)
}

// ------------- PRIVATE FUNCTIONS -------------------------------
func (w *Worker) runServices(start int, unit *service.Unit) {
	totalServices := len(w.services)
	totalAlloc := 0
	i := start

	for ; i < totalServices; i++ {
		if w.services[i] == nil {
			break
		}

		totalAlloc++
	}

	if i > totalServices {
		w.removeUnit(unit)
		unit.Promise.Resolve(unit)

		return
	}

	unit.Alloc(totalAlloc)
	unit.Done(func(unit *service.Unit) {
		w.runServices(i+1, unit)
	})

	totalServices = start + totalAlloc
	for j := start; j < totalServices; j++ {
		go w.services[j].Add(unit)
	}
}

func (w *Worker) getUnit(token *token.Token) *service.Unit {
	if unit, ok := w.running[token.ID]; ok {
		return unit
	}

	return nil
}

func (w *Worker) existUnit(unit *service.Unit) bool {
	u := w.getUnit(unit.Token)

	return u != nil
}

func (w *Worker) addUnit(unit *service.Unit) {
	w.running[unit.Token.ID] = unit
}

func (w *Worker) removeUnit(unit *service.Unit) {
	w.lock()
	delete(w.running, unit.Token.ID)
	w.unlock()
}

func (w *Worker) lock() {
	w.mutex.Lock()
}

func (w *Worker) unlock() {
	w.mutex.Unlock()
}
