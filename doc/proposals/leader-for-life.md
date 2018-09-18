# Leader for Life

## Background

Operators need leader election to ensure that if the same operator is running
in two separate pods within the same namespace, only one of them will be
active. The primary goal of leader election is to avoid contention between
multiple active operators of the same type.

High availability is not a goal of leader election for operators.

Controller-runtime is adding leader election based on functionality present [in
client-go](git@github.com:operator-framework/operator-sdk.git). However that
implementation allows for the possibility of brief periods during which
multiple leaders are active.

Requirements have been [discussed on
GitHub](https://github.com/operator-framework/operator-sdk/issues/136).

This proposal is to add leader election to the SDK that follows a "leader for
life" model, which does not allow for multiple concurrent leaders.

## Goals

* Provide leader election that is easy to use
* Provide leader election that prohibits multiple leaders

## Non-Goals

* Make operators highly available

## Solution

The "leader for life" approach uses Kubernetes features to detect when a leader
has disappeared and then automatically remove its lock. 

The approach and a PoC is detailed in [a separate
repository](https://github.com/mhrivnak/leaderelection). This proposal is to move
that implementation into `operator-sdk` and finish/modify it as appropriate.

### Usage

```golang
func main() {
    // create a lock named "myapp-lock", retrying every 5 seconds until it succeeds
    err := leader.Become("myapp-lock", 5)
    if err != nil {
        log.Fatal(err.Error())
    }
    ...
    // do whatever else your app does
}
```

## Future

Once accepted into operator-sdk, this would be valuable to contribute back to either
controller-runtime directly or client-go.
