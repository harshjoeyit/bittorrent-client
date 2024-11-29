package tracker

import (
	"math/rand"
	"time"
)

func randInt32() int32 {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	return r.Int31()
}
