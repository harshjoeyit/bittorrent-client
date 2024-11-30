// For every peer we maintain a list of blocks/pieces for which
// we have received "have" or "bitfield" message
//
// For requesting a block/piece an item is popped from front of
// the queue and request message is sent to the queue
package queue

import "github.com/gammazero/deque"

type Queue struct {
	q deque.Deque[Block]
}

func NewQueue() Queue {
	return Queue{
		q: deque.Deque[Block]{},
	}
}

// Queue methods

func (q *Queue) IsEmpty() bool {
	return q.q.Len() == 0
}

// Returns front element
func (q *Queue) Front() Block {
	return q.q.Front()
}

// Pushes to back of the queue
func (q *Queue) PushBack(b Block) {
	q.q.PushBack(b)
}

// Remove element from back of the queue
func (q *Queue) PopBack() Block {
	return q.q.PopBack()
}
