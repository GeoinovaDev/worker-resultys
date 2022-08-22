package worker

import (
	"fmt"
	"sync"
	"time"

	"github.com/GeoinovaDev/lower-resultys/time/interval"
	"github.com/GeoinovaDev/models-resultys/token"
	"github.com/GeoinovaDev/service-resultys"
	"github.com/GeoinovaDev/worker-resultys/hook"
	"github.com/GeoinovaDev/worker-resultys/queue"
)

// Worker struct
type Worker struct {
	services []service.Service
	running  map[string]*service.Unit

	unitID int
	hook   *hook.Hook

	waitQueue *queue.Queue

	timeout int

	mutex *sync.Mutex
}

// New ...
func New(timeout int) *Worker {
	w := &Worker{}

	w.mutex = &sync.Mutex{}
	w.services = []service.Service{}
	w.hook = hook.New()
	w.waitQueue = queue.New()

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
func (w *Worker) Run(unit *service.Unit, fnOnce func(*service.Unit), fnOk func(*service.Unit), fnTimeout func(*service.Unit)) *Worker {
	w.mutex.Lock()
	if w.existUnit(unit) {
		_unit := w.getUnit(unit.Token)
		_unit.Processing++
		_unit.SetStatus("Update Wait")

		w.waitQueue.Add(_unit.ID, unit)

		w.mutex.Unlock()
		return w
	}

	w.unitID++
	unit.ID = w.unitID
	unit.Processing++
	unit.SetStatus("New Wait")

	w.waitQueue.Add(unit.ID, unit)

	w.addUnit(unit)
	w.mutex.Unlock()

	interval := interval.New().Repeat(w.timeout, func() {
		unit.SetStatus("Start Interval")
		w.mutex.Lock()
		defer w.mutex.Unlock()
		unit.SetStatus("Interval Processing")

		isProcessing := w.existUnit(unit)

		if isProcessing {
			unit.SetStatus("Interval Dispatch")
			w.invoke(fnTimeout, unit)
			w.waitQueue.Clear(unit.ID)
		}

		unit.SetStatus("Finish Interval")
	})

	w.runServices(0, unit, func() {
		unit.SetStatus("Start Release")

		w.mutex.Lock()
		interval.Clear()
		w.removeUnit(unit)
		w.invoke(fnOk, unit)
		w.waitQueue.Remove(unit.ID)
		w.mutex.Unlock()

		unit.SetStatus("Finish Release")

		fnOnce(unit)
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

func (w *Worker) invoke(fn func(*service.Unit), unit *service.Unit) {
	units := w.waitQueue.Get(unit.ID)
	tokenID := unit.Token.TokenID
	webhook := unit.Token.Webhook
	webhookdID := unit.Token.WebhookID

	for i := 0; i < len(units); i++ {
		unit.Token.TokenID = units[i].Token.TokenID
		unit.Token.Webhook = units[i].Token.Webhook
		unit.Token.WebhookID = units[i].Token.WebhookID
		fn(unit)
	}

	unit.Token.TokenID = tokenID
	unit.Token.Webhook = webhook
	unit.Token.WebhookID = webhookdID
}
