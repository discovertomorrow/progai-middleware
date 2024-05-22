package llamacpp

import (
	"math/rand"
	"sync"
	"time"

	"github.com/discovertomorrow/progai-middleware/pkg/handler"
)

type Queue struct {
	n             int
	slots         []*Usage
	mutex         sync.Mutex
	semaphore     chan struct{}
	endpointSlots []EndpointSlot
}

type Slot struct {
	ID           int
	endpointSlot EndpointSlot
	last         Usage
}

type EndpointSlot struct {
	endpoint string
	slot     int
}

type Usage struct {
	user     int
	time     int64
	userSlot int
}

func NewQueue(endpoints []handler.Endpoint) *Queue {
	n := 0
	for _, ep := range endpoints {
		n += ep.Parallel
	}
	q := Queue{
		n:             n,
		slots:         make([]*Usage, n),
		semaphore:     make(chan struct{}, n),
		endpointSlots: make([]EndpointSlot, n),
	}
	s := 0
	for _, ep := range endpoints {
		for j := 0; j < ep.Parallel; j++ {
			q.endpointSlots[s] = EndpointSlot{endpoint: ep.Endpoint, slot: j}
			q.slots[s] = &Usage{user: -1, time: int64(rand.Intn(10001)), userSlot: -1}
			s += 1
		}
	}
	return &q
}

func (q *Queue) ReleaseSlot(s Slot) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.slots[s.ID] != nil {
		// handle error
	}
	u := Usage{
		user:     s.last.user,
		time:     time.Now().Unix(),
		userSlot: s.last.userSlot,
	}
	q.slots[s.ID] = &u
	<-q.semaphore
}

func (q *Queue) RequestSlot(user, userSlot int) Slot {
	q.semaphore <- struct{}{}
	q.mutex.Lock()
	defer q.mutex.Unlock()

	oldest := -1
	oldestTime := time.Now().Unix() + 1
	match := -1
	for i := range q.n {
		u := q.slots[i]
		if u == nil {
			continue
		}
		if u.user == user && u.userSlot == userSlot {
			match = i
			break
		}
		if u.time < oldestTime {
			oldestTime = u.time
			oldest = i
		}
	}
	if match == -1 {
		if oldest == -1 {
			// handle error!!!
		}
		match = oldest
	}
	eps := q.getEndpointSlot(match)
	s := Slot{
		ID:           match,
		endpointSlot: eps,
		last: Usage{
			user:     user,
			time:     0,
			userSlot: userSlot,
		},
	}
	q.slots[match] = nil
	return s
}

func (q *Queue) getEndpointSlot(id int) EndpointSlot {
	return q.endpointSlots[id]
}
