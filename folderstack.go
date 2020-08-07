package main

import (
	"container/list"
	"fmt"
)

// FolderEntry is a queued entry to track a single folder
type FolderEntry struct {
	FolderID   string // Name of the folder
	FolderPath string // Path of the containing parent
}

// FolderStack is used to maintain a queue of folders to process
// It is used as a LIFO stack so that we can end up doing an in-order traversal
type FolderStack struct {
	queue *list.List
}

// Push puts an entry at the top of the stack
func (c *FolderStack) Push(value string, parentPath string) {
	c.queue.PushFront(FolderEntry{FolderID: value, FolderPath: parentPath})
}

// Pop removes the entry from the top of the stack
func (c *FolderStack) Pop() (entry FolderEntry, err error) {
	entry, err = c.Front()

	if err == nil {
		ele := c.queue.Front()
		c.queue.Remove(ele)
	}
	return
}

// Front finds the first entry in the stack
func (c *FolderStack) Front() (FolderEntry, error) {
	if c.queue.Len() > 0 {
		if val, ok := c.queue.Front().Value.(FolderEntry); ok {
			return val, nil
		}
		return FolderEntry{}, fmt.Errorf("Peep Error: Queue Datatype is incorrect")
	}
	return FolderEntry{}, fmt.Errorf("Peep Error: Queue is empty")
}

// Size tells us how many entries there are
func (c *FolderStack) Size() int {
	return c.queue.Len()
}

// isEmpty tells when there is nothing left on the stack
func (c *FolderStack) isEmpty() bool {
	return c.queue.Len() == 0
}
