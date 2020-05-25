package rollout

import (
	"errors"
	"hash/fnv"
	"math/rand"
)

// Rollout returns true and no error when during this run something should
// happen for given actor according to the stickiness and likelihood passed
// as a percentage value to this function. It returns false rollout and an
// error if the percentage value is negative or higher than 100.
func Rollout(actor string, percentage int, stickiness string) (bool, error) {
	if percentage < 0 || percentage > 100 {
		return false, errors.New("Rollout value should be between 0 and 100 inclusive")
	}

	if percentage == 0 {
		return false, nil
	}

	if percentage == 100 {
		return true, nil
	}

	switch stickiness {
	case "random":
		return random(percentage), nil
	default:
		return forActor(actor, percentage), nil
	}
}

// random guarantees no stickiness. For every call it will yield a random
// true/false based on the provided rollout percentage.
func random(percentage int) bool {
	return rand.Intn(100) < percentage
}

// forActor provides "stickines", i.e. guarantees that the same actor
// gets the same result every time. It also assures that an actor which is
// among the first 10% will also be among the first 20%.
func forActor(actor string, percentage int) bool {
	h := fnv.New32a()
	h.Write([]byte(actor))
	sum32 := h.Sum32()

	if sum32 == 0 {
		return false
	}

	return (sum32 % uint32(100)) < uint32(percentage)
}
