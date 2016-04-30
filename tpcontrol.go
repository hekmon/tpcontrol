package tpcontrol

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// TPScheduler allows managing a global throughput while ordering requests with differents priority
type TPScheduler struct {
	prioQueues   []scQueue
	tokenPool    chan bool
	notifChannel chan bool
}

type scQueue struct {
	queue []*sync.Mutex
	mutex sync.Mutex
}

func (q *scQueue) processQueue() bool {
	// Operations on the queue are exclusives !
	q.mutex.Lock()
	defer q.mutex.Unlock()

	// Does this queue has clients waiting ?
	if len(q.queue) > 0 {
		// If it does, unlock the oldest client waiting...
		q.queue[0].Unlock()
		// ... and remove him from the queue
		q.queue = q.queue[1:]
		return true
	}
	return false
}

// New returns a fully initialized, ready to use TPScheduler object
func New(nbRequests int, nbSeconds int, nbQueues int, tokenPoolSize int) (*TPScheduler, error) {
	var sc TPScheduler

	// Pre checks
	if nbRequests < 1 {
		return nil, errors.New("nbRequests can not be lower than 1")
	}
	if nbSeconds < 1 {
		return nil, errors.New("nbSeconds can not be lower than 1")
	}
	if nbQueues < 1 {
		return nil, errors.New("You must have at least 1 queue")
	}
	if tokenPoolSize < 0 {
		return nil, errors.New("If you desire an unbuffered token pool, please set tokenPoolSize at 0")
	}

	// Init
	sc.prioQueues = make([]scQueue, nbQueues)
	sc.tokenPool = make(chan bool, tokenPoolSize)
	sc.notifChannel = make(chan bool)

	// Ticker/Seeder
	tickerDuration := (time.Duration(nbSeconds) * time.Second) / time.Duration(nbRequests)
	go func() {
		sc.tokenPool <- true // start with a token
		for range time.NewTicker(tickerDuration).C {
			sc.tokenPool <- true
		}
	}()

	// Dispatcher
	go func() {
		// When a token is available, try to unlock a client
		for range sc.tokenPool {
			// A token is ready, but do we have a client waiting ?
			<-sc.notifChannel
			// When we do, find the most important client and release its lock
			i := 0
			for ; i < len(sc.prioQueues); i++ {
				if sc.prioQueues[i].processQueue() {
					break // processQueue() did release a client lock, the current token is now consumed
				}
			}
			// If we were not able to find a client, that's a bug, 1 write in sc.notifChannel == 1 mutex created in a queue !
			if i == len(sc.prioQueues) {
				panic("A ghost, there is a ghost ! I'm too afraid to continue to work... (dispatcher broken)")
			}
		}
	}()

	return &sc, nil
}

// CanIGO blocks the caller until the dispatcher confirms it is ok to proceed
// priority is the queue index to use for registration (0 being the highest priority)
func (sc *TPScheduler) CanIGO(priority int) error {

	// Does the queue exist ?
	if priority < 0 {
		return errors.New("Priority level can't be lower than 0")
	}
	if priority+1 > len(sc.prioQueues) {
		errorMsg := fmt.Sprintf("Priority level %d does not exist : you only have %d priority queues, therefor the lowest possible priority is %d",
			priority, len(sc.prioQueues), len(sc.prioQueues)-1)
		return errors.New(errorMsg)
	}

	// Prepare client lock
	var clientLock sync.Mutex
	clientLock.Lock()

	// Register it (in order please)
	sc.prioQueues[priority].mutex.Lock()
	sc.prioQueues[priority].queue = append(sc.prioQueues[priority].queue, &clientLock)
	sc.prioQueues[priority].mutex.Unlock()

	// Tell the dispatcher we have a client waiting (the notification must not be blocking, so we launch it in a goroutine)
	go func() {
		sc.notifChannel <- true // this will wake up/unlock the dispatcher (if/when a token is available)
	}()

	// Hold on the client lock and return when dispatcher unlock us
	clientLock.Lock()
	return nil
}
