# TPControl
[![GoDoc](https://godoc.org/github.com/Hekmon/TPControl?status.svg)](https://godoc.org/github.com/Hekmon/TPControl)

TPControl provides a simple yet powerfull scheduler which allows to manage a given throughput for any number of workers and optionally allows to prioritize their requests as well.

# Use case

For everything you want. But in my case, I developped this package for a specific need : to keep in check API requests on an external service.

Usually with external/public API, you have to maintain a maximal throughput in order to avoid getting your key or account temporarily or permanently banned. By spawning a TPControl.Scheduler and setting the throughput you need, you will be assured that every worker asking the scheduler if it can start will only begin when the defined throughput allows it.

Also, when your service is becoming big, you might have severals differents process each with different priority making all several requests in parallel. TPControl offer priority queues management in order to maintain your throughput in any case but by unlocking most important workers first.

# Usage

Every example below is from (and can be try with) the example source code [here](https://github.com/Hekmon/TPControl/blob/master/example/tpcontrol_example.go).

There is 3 differents usages/examples :

* [Simple throughput manager](https://github.com/Hekmon/TPControl#simple-throughput-manager)
* [Simple throughput manager with a token pool/buffer](https://github.com/Hekmon/TPControl#simple-throughput-manager-with-a-token-poolbuffer)
* [Advanced throughput manager with priority management](https://github.com/Hekmon/TPControl#advanced-throughput-manager-with-priority-management)

Or may be you are just interested on how is it working ? In that case, you can jump right [here](https://github.com/Hekmon/TPControl#how-does-it-work-).

## Simple throughput manager

For the first example, let's keep it simple : we want a scheduler for 5 requests per second. And that's it. No cache, no burst and no priority stuff.

Instanciating this scheduler would look like this :
```go
scheduler, err := tpcontrol.New(5, 1, 1, 0)
if err != nil {
	panic(err)
}
```

The first two parameters represent the desired throughput : 5 requests over 1 second. For a simple throughput manager, leave the last 2 parameters at : 1 (for nbQueues) and 0 (for tokenPoolSize). More details on [godoc](https://godoc.org/github.com/Hekmon/TPControl#New).

To hook on the scheduler, the only thing a worker need to do is the following call :
```go
scheduler.CanIGO(0)
```

This call will block until the scheduler says it is ok to perform. Notice the `0` parameter. It is indicating the priority queue. As we don't care for priority right now, we spawned our scheduler with only 1 queue, so we must use the only one existing, the highest priority queue which has for index : `0`.

Using these scheduler parameters with the [example](https://github.com/Hekmon/TPControl/blob/master/example/tpcontrol_example.go) will output :
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

As you can see, each worker started working with a 200ms gap, respecting our throughput of 5 requests per second.

For those wondering why batch numbers are not in order (you are right, the scheduler's queues are FIFO !) this is only related on how GO manages and starts goroutines. We created them in order, GO started them how it wanted ;)


## Simple throughput manager with a token pool/buffer

But sometimes, your app won't send any requests during a certain amount of time, so why not take advantage of it and allow a burst when new requests will arrive after that ? This should not affect your global throughput if set up correctly (especially if the throughput is specified over a time range, like 5 minutes).

Let's keep our last example of 5 req/s but this time, let's say that if we did not sent any requests for the last second (so... 5 requests) we allow ourself to use them anyway. This is our token pool.

The scheduler would be instanciated like this :
```go
scheduler, err := tpcontrol.New(5, 1, 1, 5)
if err != nil {
	panic(err)
}
```
So here we have : 5 requests over 1 second, 1 queue and 5 for the token pool size. Don't hesitate to check [godoc](https://godoc.org/github.com/Hekmon/TPControl#New).

For the [example](https://github.com/Hekmon/TPControl/blob/master/example/tpcontrol_example.go), let's rise the number of batches up to 10 :
```
The token pool size is 5, let's wait 1s to let it fill up completly (based on flow defined as 5.00 req/s).
Time's up !

10 workers launched...

I am a worker with a priority of 0 coming from the batch 4 and this experiment started 500.9µs ago.
I am a worker with a priority of 0 coming from the batch 0 and this experiment started 500.9µs ago.
I am a worker with a priority of 0 coming from the batch 1 and this experiment started 500.9µs ago.
I am a worker with a priority of 0 coming from the batch 2 and this experiment started 500.9µs ago.
I am a worker with a priority of 0 coming from the batch 3 and this experiment started 500.9µs ago.
I am a worker with a priority of 0 coming from the batch 5 and this experiment started 199.1415ms ago.
I am a worker with a priority of 0 coming from the batch 7 and this experiment started 399.2836ms ago.
I am a worker with a priority of 0 coming from the batch 6 and this experiment started 598.6465ms ago.
I am a worker with a priority of 0 coming from the batch 8 and this experiment started 798.7885ms ago.
I am a worker with a priority of 0 coming from the batch 9 and this experiment started 998.9306ms ago.

10 workers ended their work.
```

The demo program wait the right time to let the pool fill itself up and have its maximum token capacity available.

As you can see the first 5 requests used the tokens in the storage pool to execute themself right away. Then, the storage pool was depleted and the others workers had to wait the new generated tokens to continue their execution. New tokens are still generated in order to respect the given throughput (200ms).


## Advanced throughput manager with priority management

This time, let's say we have 3 differents process each less important than the other. Every request coming from `n` should be treated before requests from `n+1` and each requests coming from `n+1` should be treated before n+2. These are our priority queues.

Keeping our throughput of 5 requests per second (but with no token pool for now) we would instanciate the scheduler like this :
```go
scheduler, err := tpcontrol.New(5, 1, 3, 0)
if err != nil {
	panic(err)
}
```
Remember, [godoc](https://godoc.org/github.com/Hekmon/TPControl#New) is your friend.

So if a worker wants to register itself with the lowest priority :
```go
scheduler.CanIGO(2)
```

As `2` is the index of the queue, it will register itself on the third queue (the lowest priority in this case).

Each batch will create a worker for each queue : one high priority (0), one medium priority (1) and one low priority (2). Of course each process can make/create several concurrent requests and they should be treated as FIFO for a given priority. Let's run the [example](https://github.com/Hekmon/TPControl/blob/master/example/tpcontrol_example.go) with 3 batches :
```
The token pool size is 0, let's wait 0 to let it fill up completly (based on flow defined as 5.00 req/s).
Time's up !

9 workers launched...

I am a worker with a priority of 0 coming from the batch 0 and this experiment started 199.6418ms ago.
I am a worker with a priority of 0 coming from the batch 1 and this experiment started 399.7835ms ago.
I am a worker with a priority of 0 coming from the batch 2 and this experiment started 599.9282ms ago.
I am a worker with a priority of 1 coming from the batch 0 and this experiment started 799.07ms ago.
I am a worker with a priority of 1 coming from the batch 1 and this experiment started 999.4399ms ago.
I am a worker with a priority of 1 coming from the batch 2 and this experiment started 1.1995819s ago.
I am a worker with a priority of 2 coming from the batch 0 and this experiment started 1.3997237s ago.
I am a worker with a priority of 2 coming from the batch 1 and this experiment started 1.599866s ago.
I am a worker with a priority of 2 coming from the batch 2 and this experiment started 1.7990072s ago.

9 workers ended their work.
```

The important thing here is that the first 3 workers allowed to perform here were the priority 0 workers from each different batch. Then the priority 1. Then 2. Priority has been respected, so did throughput.


So... A last one with a token pool ? :3

Let's go for a 10 requests over 3 seconds with 3 different priority queues and a token pool of 5 (because why not) :
```go
scheduler, err := tpcontrol.New(10, 3, 3, 5)
if err != nil {
	panic(err)
}
```

With a 4 batches run :
```
The token pool size is 5, let's wait 1.5s to let it fill up completly (based on flow defined as 3.33 req/s).
Time's up !

12 workers launched...

I am a worker with a priority of 2 coming from the batch 1 and this experiment started 500.6µs ago.
I am a worker with a priority of 0 coming from the batch 0 and this experiment started 500.6µs ago.
I am a worker with a priority of 1 coming from the batch 0 and this experiment started 500.6µs ago.
I am a worker with a priority of 2 coming from the batch 0 and this experiment started 500.6µs ago.
I am a worker with a priority of 0 coming from the batch 1 and this experiment started 500.6µs ago.
I am a worker with a priority of 0 coming from the batch 2 and this experiment started 299.2146ms ago.
I am a worker with a priority of 0 coming from the batch 3 and this experiment started 599.4277ms ago.
I am a worker with a priority of 1 coming from the batch 1 and this experiment started 898.8548ms ago.
I am a worker with a priority of 1 coming from the batch 2 and this experiment started 1.1990676s ago.
I am a worker with a priority of 1 coming from the batch 3 and this experiment started 1.4992807s ago.
I am a worker with a priority of 2 coming from the batch 2 and this experiment started 1.7987152s ago.
I am a worker with a priority of 2 coming from the batch 3 and this experiment started 2.0994297s ago.

12 workers ended their work.
```

Here you might wonder why the first requests (using the buffered tokens in the pool) are not ordered. This is the same reason as for the "Simple throughput manager" example : we did declare all the workers sequentially but GO decided on its own which one to start first. And as the first ones ask the scheduler, there were no wait, tokens were availables, scheduler unlock them almost instantly.

Once the token pool depleted, workers registered but were not unlock instantly and when a token became available, the scheduler unlocked the oldest one in the highest priority queue (`I am a worker with a priority of 0 coming from the batch 2 and this experiment started 299.2146ms ago.`).


# How does it work ?

## If you does not mind reading

The TPControl.Scheduler is composed by severals components :

* A seeder
* A token pool buffer
* A dispatcher
* A notification channel
* A blocking registration method  ( the `CanIGO(priority)` )

### The seeder

Mostly composed by a ticker set up with the throughput indicated with the two first parameters of the `New()` function : `nbRequests` and `nbSeconds`. The seeder runs on its own goroutine and at every tick, it will generate a token and try to put it on the pool. If the pool is full (or just unbuffered, aka at 0) the seeder waits to be able to drop the new token into the pool, putting itself to sleep.

### The token pool buffer

This one is a go channel which can be buffered or unbuffured depending on the parameter `tokenPoolSize` from the `New()` function. It allows token storage in case of a buffered setup but mostly allows communication between the seeder and the dispatcher.

### The dispatcher

When reading a token from the token pool buffer, the dispatcher will immediately check if a client notification is present (more on that just after). When it reads a notification, the dispatcher will look for the oldest client in the highest priority queue in order to unlock it. If no client notification is present, the dispatcher wait for one (channel read) and do not consumme tokens anymore, putting the dispatcher and the seeder (if the token pool is unbuffered or full) on sleep. Similarly, if there is clients notifications pending, but no tokens available, the dispatcher waits (another channel read) one before proceeding, putting itself to sleep in that case too.

### The notification channel

This is part of the main trick. This channel is unbuffered and allows to not have the dispatcher going postal on an infite loop while checking every queue all the time : while empty, the dispatcher will block on reading it. When a client (or many) register for execution, they send (more on that just after) a notification through this channel to wake up the scheduler (if it was locked on the notification chan and not the token pool of course).

### The blocking registration method

Finally. When a worker calls this method, 3 things happen :

* A lock is generated and registrated for this client
* A notification is send asynchronously to the dispatcher
* The method blocks until the dispatcher unlocks it

The lock is a simple mutex spawned and pre-locked added to the priority queue passed as parameter of the method.

Then the method must inform the dispatcher that a client is waiting and for that a write inside the notification channel is needed. But if the method try to make that write synchronously, the method might hangs because others workers would be registrating in the same time. Imagine another worker with a lesser priority but already registered trying to make its own notification : by locking the channel with its write, we might spend to much time trying too make our notification while the dispatcher would already have unlocked our client lock. Making this channel buffered to overcome this limitation is also not a good idea : how to determine the/a good buffer length ? The solution is simple : launch the notification in another goroutine (they are cheap !) to make the notification non blocking.

This way the registration method can get to the third point : holding up on its personnal lock and be reading for the dispatcher to unlock it and finally return to the caller.

In fact (and this is the other part of the main trick) we no dot care if this is our notification which woke up the dispatcher just before it unlocked us : as long as there is as many notifications waiting as client locks registrated, this is ok : all we need is a dispatcher awaken when clients are waiting and asleep when no one is here.

## TL;DR (or maybe you also want the big picture ?)

[![TPControl schematic](https://raw.githubusercontent.com/Hekmon/TPControl/master/tpcontrol.png)](https://raw.githubusercontent.com/Hekmon/TPControl/master/tpcontrol.png)

# License

MIT licensed. See the LICENSE file for details.