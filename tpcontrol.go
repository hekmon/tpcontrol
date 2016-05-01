package tpcontrol

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// TPScheduler allows managing a global throughput while ordering requests with differents priority
type TPScheduler struct {
	prioQueues            []scQueue
	seeder                scSeeder
	dispatcherRunningFlag sync.Mutex
	tokenPool             chan bool
	notifChannel          chan bool
}

// Scheduler Queue subtype and process method
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

// Scheduler seeder subtype
type scSeeder struct {
	ticker      *time.Ticker
	stopSignal  chan bool		// closing this channel will be interpreted by the seeder as a close instruction
	runningFlag sync.Mutex
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

	// Seeder
	tickerDuration := (time.Duration(nbSeconds) * time.Second) / time.Duration(nbRequests)
	sc.seeder.ticker = time.NewTicker(tickerDuration)
	sc.seeder.stopSignal = make(chan bool)
	go func() {
		sc.seeder.runningFlag.Lock() // mark the seeder as running (used in Stop())
		for {
			select {
			case <-sc.seeder.ticker.C: 
				sc.tokenPool <- true // Create/Add a new token in the pool
			case <-sc.seeder.stopSignal:
				// Stop the ticker and dereference it (for GC to collect it)
				sc.seeder.ticker.Stop()
				sc.seeder.ticker = nil
				// Once the ticker is stopped, we can safely close the tokenPool and mark the seeder as closed
				close(sc.tokenPool)
				sc.seeder.runningFlag.Unlock()
				fmt.Println("Seeder ended")
				return
			}
		}
	}()

	// Dispatcher
	go func() {
		sc.dispatcherRunningFlag.Lock() // First mark the dispatcher as running
		// When a token is available, try to unlock a client
		for range sc.tokenPool {
			// A token is ready, but do we have a client waiting ?
			if _, channelOpen := <-sc.notifChannel; channelOpen {
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
		}
		// unlock the running flag (used by Stop())
		sc.dispatcherRunningFlag.Unlock()
		fmt.Println("Dispatcher ended")
	}()

	return &sc, nil
}


// CanIGO blocks the caller until the dispatcher confirms it is ok to proceed.
// Priority parameter is the queue index to use for registration (0 being the highest priority)
func (sc *TPScheduler) CanIGO(priority int) error {

	// Is the scheduler running ?
	if sc.seeder.ticker == nil {
		return errors.New("TPScheduler is not running (anymore ?)")
	}

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


// Stop ends the dispatcher and the seeder goroutines of the TPScheduler. It also unlocks all the queues.
// Best is to be sure to not call canIGO() anymore once the Stop() called in case one slip in during the stop process.
// After the call, the GC should be able to clean the TPScheduler entirely if unreferenced.
func (sc *TPScheduler) Stop() {
	fmt.Println("Stopping the scheduler...")

	// Send the signal to stop the seeder
	// It will stop and dereference the seeder's ticker but most importantly, close the tokenPool
	close(sc.seeder.stopSignal)

	// In order to stop the dispatcher, both the tokenPool and the notifChannel must be closed.
	// The tokenPool will be closed by the seeder, so let's close the notifChannel ourself.
	close(sc.notifChannel)

	// Wait for both seeder and dispatcher to end
	sc.seeder.runningFlag.Lock()
	sc.dispatcherRunningFlag.Lock()

	// Now that both independant process (seeder and dispatcher) have been stopped, purge/release all the queues
	for currentQueue := 0 ; currentQueue < len(sc.prioQueues) ; currentQueue++ {
		for sc.prioQueues[currentQueue].processQueue() {}
	}

	fmt.Println("Scheduler stopped !")
}
