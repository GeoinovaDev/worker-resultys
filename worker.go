package worker

import (
	"fmt"
	"sync"
	"time"

	"git.resultys.com.br/lib/lower/time/interval"
	"git.resultys.com.br/motor/models/token"
	"git.resultys.com.br/motor/service"
	"git.resultys.com.br/motor/worker/hook"
)

// Worker struct
type Worker struct {
	services []service.Service
	running  map[string]*service.Unit

	unitID int
	hook   *hook.Hook

	timeout int

	mutex *sync.Mutex
}

// New ...
func New(timeout int) *Worker {
	w := &Worker{}

	w.mutex = &sync.Mutex{}
	w.services = []service.Service{}
	w.hook = hook.New()

	w.timeout = timeout

	w.running = make(map[string]*service.Unit)

	return w
}

// SetTimeout ...
func (w *Worker) SetTimeout(t int) *Worker {
	w.timeout = t

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
func (w *Worker) Run(unit *service.Unit, once func(*service.Unit), ok func(*service.Unit), timeout func(*service.Unit)) *Worker {
	w.mutex.Lock()
	if w.existUnit(unit) {
		unit = w.getUnit(unit.Token)
		unit.Tag = "Update Wait"

		w.hook.On("ok:"+unit.GetUUID(), ok)
		w.hook.On("timeout:"+unit.GetUUID(), timeout)

		w.mutex.Unlock()
		return w
	}

	w.unitID++
	unit.ID = w.unitID
	unit.Tag = "New Wait"

	w.hook.On("ok:"+unit.GetUUID(), ok)
	w.hook.On("once:"+unit.GetUUID(), once)
	w.hook.On("timeout:"+unit.GetUUID(), timeout)

	w.addUnit(unit)
	w.mutex.Unlock()

	interval := interval.New().Repeat(w.timeout, func() {
		w.mutex.Lock()
		defer w.mutex.Unlock()

		isProcessing := w.existUnit(unit)

		if isProcessing {
			w.hook.Trigger("timeout:"+unit.GetUUID(), unit)
			w.hook.Off("ok:" + unit.GetUUID())
			w.hook.Off("timeout:" + unit.GetUUID())
		}
	})

	w.runServices(0, unit, func() {
		w.mutex.Lock()
		defer w.mutex.Unlock()

		interval.Clear()

		w.removeUnit(unit)
		w.hook.Trigger("ok:"+unit.GetUUID(), unit)
		w.hook.Trigger("once:"+unit.GetUUID(), unit)

		w.hook.Off("ok:" + unit.GetUUID())
		w.hook.Off("once:" + unit.GetUUID())
		w.hook.Off("timeout:" + unit.GetUUID())
	})

	return w
}

// Exist ...
func (w *Worker) Exist(unit *service.Unit) bool {
	w.mutex.Lock()
	u := w.getUnit(unit.Token)
	w.mutex.Unlock()

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

// Running ...
func (w *Worker) Running() []*service.Unit {
	arr := []*service.Unit{}

	// w.mutex.Lock()
	// defer w.mutex.Unlock()

	for name := range w.running {
		arr = append(arr, w.running[name])
	}

	return arr
}

// ------------- PRIVATE FUNCTIONS -------------------------------
func (w *Worker) runServices(start int, unit *service.Unit, done func()) {
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
		go done()
		return
	}

	unit.Alloc(totalAlloc)
	unit.Done(func(unit *service.Unit) {
		w.runServices(i+1, unit, done)
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
	delete(w.running, unit.Token.ID)
}
