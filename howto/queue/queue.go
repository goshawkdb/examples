// {{define "6d377a2e-4684-4448-8ca7-39c18162e2e8"}}
package queue

import (
	"errors"
	"goshawkdb.io/client"
)

type Queue struct {
	objRef client.RefCap
}

const (
	queueHead = 0
	queueTail = 1

	cellValue = 0
	cellNext  = 1
)

func NewQueue(txr client.Transactor) (*Queue, error) {
	result, err := txr.Transact(func(txn *client.Transaction) (interface{}, error) {
		rootObj, found := txn.Root("myRoot1")
		if !found {
			return nil, errors.New("No root 'myRoot1' found")
		}
		err := txn.Write(rootObj, []byte{})
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}
		return rootObj, nil
	})
	if err != nil {
		return nil, err
	}
	return &Queue{
		objRef: result.(client.RefCap),
	}, nil
}

func (q *Queue) Append(txr client.Transactor, item client.RefCap) error {
	_, err := txr.Transact(func(txn *client.Transaction) (interface{}, error) {
		// create a new cell that only has a ref to the value being appended.
		cell, err := txn.Create([]byte{}, item)
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}
		_, queueRefs, err := txn.Read(q.objRef)
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}
		if len(queueRefs) == 0 {
			// queue is completely empty: set both head and tail at once.
			return nil, txn.Write(q.objRef, []byte{}, cell, cell)

		} else {
			tailCell := queueRefs[queueTail]
			_, tailCellRefs, err := txn.Read(tailCell)
			if err != nil || txn.RestartNeeded() {
				return nil, err
			}
			// append our new cell to the refs of the current tail
			err = txn.Write(tailCell, []byte{}, append(tailCellRefs, cell)...)
			if err != nil || txn.RestartNeeded() {
				return nil, err
			}
			// update the queue tail to point at the new tail.
			queueRefs[queueTail] = cell
			return nil, txn.Write(q.objRef, []byte{}, queueRefs...)
		}
	})
	return err
}

// {{end}}
// {{define "6dee71c7-41e8-4b3a-8a62-f92d1444bd2e"}}
func (q *Queue) Dequeue(txr client.Transactor) (client.RefCap, error) {
	result, err := txr.Transact(func(txn *client.Transaction) (interface{}, error) {
		_, queueRefs, err := txn.Read(q.objRef)
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}
		if len(queueRefs) == 0 {
			// queue is completely empty; Let's wait until it's not!
			return nil, txn.Retry()

		} else {
			headCell := queueRefs[queueHead]
			_, headCellRefs, err := txn.Read(headCell)
			if err != nil || txn.RestartNeeded() {
				return nil, err
			}
			item := headCellRefs[cellValue]
			if len(headCellRefs) == 1 {
				// there's only one item in the queue and we've just
				// consumed it, so remove all references from the queue
				return item, txn.Write(q.objRef, []byte{})

			} else {
				// the queue head should point to the next cell. The queue
				// tail doesn't change.
				queueRefs[queueHead] = headCellRefs[cellNext]
				return item, txn.Write(q.objRef, []byte{}, queueRefs...)
			}
		}
	})
	if err != nil {
		return client.RefCap{}, err
	}
	return result.(client.RefCap), nil
}

// {{end}}
