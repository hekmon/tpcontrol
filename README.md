# TPControl
[![GoDoc](https://godoc.org/github.com/Hekmon/TPControl?status.svg)](https://godoc.org/github.com/Hekmon/TPControl)

TPControl provide a simple yet powerfull scheduler which allows to manage a given throughput for any number of workers and optionally allows to prioritize their requests as well.

# Use case

For everything you want. But in my case, I developped this package for a specific need : to keep in check API requests on an external service.

Usually with external/public API, you have to maintain a maximal throughput in order to not get your API key or account banned (temporarily or permanently). By spawning a TPScheduler and setting the throughput you need, you will be assured that every worker asking the scheduler if it can start will only begin when the defined throughput allows it.

Also, when your service is becoming big, you might have severals differents process each with different priority making all several requests in parallel. TPControl offer priority queues management in order to maintain your throughput in any case but by unlocking more important workers first.

# Usage

Every example below is from (and can be try with) the example source code [here](https://github.com/Hekmon/TPControl/blob/master/example/tpcontrol_example.go).

## Simple throughput manager

For the first example, let's keep it simple : we want a scheduler for 5 requests per second. And that's it. No cache, no burst no priority.

Instanciating this scheduler should look like this :
```go
scheduler, err := tpcontrol.New(5, 1, 1, 0)
if err != nil {
	panic(err)
}
```

The first two parameters represent the desired throughput : 5 requests on 1 second. For a simple throughput manager, leave the last 2 at 1 (for nbQueues) and 0 (for tokenPoolSize).

To hook on the scheduler, the only thing a worker need to do is the following call :
```go
scheduler.CanIGO(0)
```

This call will block until the scheduler says it is ok to perform. Notice the `0` parameter. It is indicating the priority queue. But as we don't care for priority right now and spawned our scheduler with only 1 queue, we can only use the only one existing, the highest priority : `0`.

Using these sceduler parameters with the [example](https://github.com/Hekmon/TPControl/blob/master/example/tpcontrol_example.go) will output :
```
The token pool size is 0, let's wait 0 to let it fill up completly (based on flow defined as 5.00 req/s).
Time's up !

5 workers launched...

I am a worker with a priority of 0 coming from the batch 1 and this experiment started 200.142ms ago.
I am a worker with a priority of 0 coming from the batch 0 and this experiment started 400.2867ms ago.
I am a worker with a priority of 0 coming from the batch 2 and this experiment started 600.4337ms ago.
I am a worker with a priority of 0 coming from the batch 3 and this experiment started 799.8043ms ago.
I am a worker with a priority of 0 coming from the batch 4 and this experiment started 999.9516ms ago.

5 workers ended their work.
```

As you can see, each worker started working with 200ms difference, respecting our throughput of 5 requests per second. 


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

This is part of the main trick. This channel is unbuffered and allows to not have the dispatcher going postal on an infite loop while checking every queue all the time : while empty the dispatcher will block on reading it. When a client (or many) register for  execution, they send (more on that just after) a notification through this channel to wake up the scheduler (if a token is available of course).

### The blocking registration method

Finally. When a worker calls this method, 3 things happen :

* A lock is generated and registrated for this client
* A notification is send asynchronously to the dispatcher
* The method hangs until the dispatcher unlocks it

The lock is a simple mutex spawned and pre-locked added to the priority queue passed as parameter of the method.

Then the method must inform the dispatcher that a client is waiting, for that a write inside the notification channel is needed. But if the method try to make that write synchronously, the method might hangs because others workers might be registrating in the same time. Imagine another worker with a lesser priority but already registered trying to make its own notification : by locking the channel with its write, we might spend to much time trying to make ours while the dispatcher already unlocked our client lock. Making this channel buffered to overcome this limitation is also not a good idea : how to determine the/a good buffer length ? The solution is simple : launch the notification in another goroutine (they are cheap !).

This way the registration method can get to the third point : holding up on its personnal lock and be reading for the dispatcher to unlock it.

In fact (and this is the other part of the main trick) we no dot care if this is our notification which woke up the dispatcher just before it unlocked us : as long as there is as many notifications waiting as client lock registrated, this is ok : all we need is a dispatcher awaken when clients are working and asleep when no one is waiting.

## TL;DR (or maybe you also want the big picture ?)

[![TPControl schematic](https://raw.githubusercontent.com/Hekmon/TPControl/master/tpcontrol.png)](https://raw.githubusercontent.com/Hekmon/TPControl/master/tpcontrol.png)

# License

MIT licensed. See the LICENSE file for details.