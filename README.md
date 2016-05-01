# TPControl
[![GoDoc](https://godoc.org/github.com/Hekmon/TPControl?status.svg)](https://godoc.org/github.com/Hekmon/TPControl)

TPControl provide a simple yet powerfull scheduler which allows to manage a given throughput for any number of workers and optionally allows to prioritize their requests as well.

# Use case

For everything you want. But in my case, I developped this package for a specific need : to keep in check API requests on an external service.

Usually with external/public API, you have to maintain a maximal throughput in order to not get your API key or account banned (temporarily or permanently). By spawning a TPScheduler and setting the throughput you need, you will be assured that every worker asking the scheduler if it can start will only begin when the defined throughput allows it.

Also, when your service is becoming big, you might have severals differents process each with different priority making all several requests in parallel. TPControl offer priority queues management in order to maintain your throughput in any case but by unlocking more important workers first.

# Usage

## Simple throughput manager

`TODO`

## Simple throughput manager with a token pool/buffer

`TODO`

## Advanced throughput manager with priority management and a token pool

`TODO`

# How does it work ?

## For readers

The TPScheduler is composed by severals components :

* A seeder
* A token pool buffer
* A dispatcher
* A notification channel
* A blocking registration method  ( the CanIGO(priority) one )

### The seeder

Mostly composed by a ticker set up with the throughput indicated with the two first parameters of the `New()` function : `nbRequests` and `nbSeconds`. The seeder runs on its own goroutine and everytick, it will generate a token a put it on the pool. If the pool is full (or just unbuffered, we will see that just after). The seeder waits to be able to drop the new token into the pool, putting it to sleep.

### The token pool buffer

This one is a go channel which can be buffered or unbuffured depending on the parameter `tokenPoolSize` from the `New()` function. It allows tokens storage in case of a buffered setup but mostly allows communication between the seeder and the dispatcher.

### The dispatcher

When reading a token from the token pool buffer, the dispatcher will immediately check if a client notification is present (more on that just after). When it reads a notification, the dispatcher will look for the oldest client in the highest priority queue in order to unlock it waiting for a new token to be available. If no client notification is present, the dispatcher wait for one (channel read) and do not consumme tokens anymore, putting the dispatcher and the seeder (if the token pool is unbuffered or full) on sleep.

### The notification channel

This is part of the main trick. This channel is unbuffered and allows to not have the dispatcher going postal on an infite loop while checking every queue all the time : while empty the dispatcher will block on reading it. When a client (or many) register for  execution, they send (more on that just after) a notification through this channel to wake up the sceduler (if a token is available of course).

### The blocking registration method

Finally. When a worker calls this method, 3 things happen :

* A lock is generated and registrated for this client
* A notification is send asynchronously to the dispatcher
* The method hangs until the dispatcher unlocks it

The lock is a simple mutex spawned and pre-locked added to the priority queue passed as parameter of the method.

Then the method must inform the dispatcher that a client is waiting, for that a write inside the notification channel is needed. But if the method try to make that write synchronously, the method might hangs because others workers might be registrating in the same time. Imagine another worker with a lesser priority but already registered : by locking the channel with its write, we might spend to much time trying to make that notification while the dispatcher already unlocked our client lock. Making this channel buffered to overcome this limitation is also not a good idea : how to determine the/a good buffer length ? The solution is simple : launch the notification in another goroutine (they are cheap !).

This way the registration method can get to the third point : holding up on its personnal lock and be reading for the dispatcher to unlock it.

In fact (and this is the other part of the main trick) we no dot care if this is our notification which woke up the dispatcher just before it unlocked us : as long as there is as many notifications waiting as client lock registrated, this is ok : all we need is a dispatcher awaken when clients are working and asleep when no one is waiting.

## TL;DR (or maybe you also want the big picture ?)

[![TPControl schematic](https://raw.githubusercontent.com/Hekmon/TPControl/master/tpcontrol.png)](https://raw.githubusercontent.com/Hekmon/TPControl/master/tpcontrol.png)

# License

MIT licensed. See the LICENSE file for details.