package valuecounter

import (
	"container/heap"
)

type ValueCounter struct {
	pq     pQueue
	counts map[int]int
}

func New() *ValueCounter {
	return &ValueCounter{
		pq:     make(pQueue, 0),
		counts: make(map[int]int),
	}
}

func (c *ValueCounter) IncFor(val int) {
	c.counts[val]++
	c.updatePq(val)
}

func (c *ValueCounter) MostCounted() (v int, ok bool) {
	if len(c.pq) == 0 {
		return 0, false
	}
	return c.pq[0].value, true
}

func (c *ValueCounter) LeastCounted() (v int, ok bool) {
	if len(c.pq) == 0 {
		return 0, false
	}
	return c.pq[len(c.pq)-1].value, true
}

func (c *ValueCounter) updatePq(val int) {

	if count, ok := c.counts[val]; ok {
		item := &pqItem{value: val, priority: count}
		if count < len(c.pq) && c.pq[count].priority == count {
			heap.Remove(&c.pq, count)
		}
		heap.Push(&c.pq, item)
		return
	}

	c.counts[val] = 1
	item := &pqItem{value: val, priority: 1}
	heap.Push(&c.pq, item)
}
