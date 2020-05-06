package queue

import "git.resultys.com.br/motor/service"

// Queue ...
type Queue struct {
	items map[int][]*service.Unit
}

// New ...
func New() *Queue {
	return &Queue{items: map[int][]*service.Unit{}}
}

// Add ...
func (q *Queue) Add(id int, unit *service.Unit) *Queue {
	if !q.Exist(id) {
		q.Clear(id)
	}

	q.items[id] = append(q.items[id], unit)

	return q
}

// Clear ...
func (q *Queue) Clear(id int) *Queue {
	q.items[id] = []*service.Unit{}

	return q
}

// Remove ...
func (q *Queue) Remove(id int) *Queue {
	if q.Exist(id) {
		q.Clear(id)
		delete(q.items, id)
	}

	return q
}

// Get ...
func (q *Queue) Get(id int) []*service.Unit {
	if q.Exist(id) {
		return q.items[id]
	}

	return []*service.Unit{}
}

// Exist ...
func (q *Queue) Exist(id int) bool {
	if _, ok := q.items[id]; ok {
		return true
	}

	return false
}
