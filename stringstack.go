package main

import (
	"container/list"
	"fmt"
)

// folderStack is used to maintain a queue of folders to process
// It is used as a LIFO stack so that we can end up doing an in-order traversal
type folderStack struct {
	queue *list.List
}

// Push puts an entry at the top of the stack
func (c *folderStack) Push(value string) {
	c.queue.PushFront(value)
}

// Pop removes the entry from the top of the stack
func (c *folderStack) Pop() (val string, err error) {
	val, err = c.Front()

	if err == nil {
		ele := c.queue.Front()
		c.queue.Remove(ele)
	}
	return
}

// Front finds the first entry in the stack
func (c *folderStack) Front() (string, error) {
	if c.queue.Len() > 0 {
		if val, ok := c.queue.Front().Value.(string); ok {
			return val, nil
		}
		return "", fmt.Errorf("Peep Error: Queue Datatype is incorrect")
	}
	return "", fmt.Errorf("Peep Error: Queue is empty")
}

// Size tells us how many entries there are
func (c *folderStack) Size() int {
	return c.queue.Len()
}

// isEmpty tells when there is nothing left on the stack
func (c *folderStack) isEmpty() bool {
	return c.queue.Len() == 0
}
