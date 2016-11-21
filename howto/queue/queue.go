// {{define "6d377a2e-4684-4448-8ca7-39c18162e2e8"}}
package queue

import (
	"goshawkdb.io/client"
)

type Queue struct {
	conn   *client.Connection
	objRef client.ObjectRef
}

const (
	queueHead = 0
	queueTail = 1

	cellValue = 0
	cellNext  = 1
)

func NewQueue(conn *client.Connection) (*Queue, error) {
	result, _, err := conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
		roots, err := txn.GetRootObjects()
		if err != nil {
			return nil, err
		}
		rootObj := roots["myRoot1"]
		err = rootObj.Set([]byte{})
		if err != nil {
			return nil, err
		}
		return rootObj, nil
	})
	if err != nil {
		return nil, err
	}
	return &Queue{
		conn:   conn,
		objRef: result.(client.ObjectRef),
	}, nil
}

func (q *Queue) Append(item client.ObjectRef) error {
	_, _, err := q.conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
		queue, err := txn.GetObject(q.objRef)
		if err != nil {
			return nil, err
		}
		// create a new cell that only has a ref to the value being appended.
		cell, err := txn.CreateObject([]byte{}, item)
		if err != nil {
			return nil, err
		}
		queueReferences, err := queue.References()
		if err != nil {
			return nil, err
		}
		if len(queueReferences) == 0 {
			// queue is completely empty: set both head and tail at once.
			return nil, queue.Set([]byte{}, cell, cell)

		} else {
			tailCell := queueReferences[queueTail]
			tailCellReferences, err := tailCell.References()
			if err != nil {
				return nil, err
			}
			// append our new cell to the refs of the current tail
			err = tailCell.Set([]byte{}, append(tailCellReferences, cell)...)
			if err != nil {
				return nil, err
			}
			// update the queue tail to point at the new tail.
			queueReferences[queueTail] = cell
			return nil, queue.Set([]byte{}, queueReferences...)
		}
	})
	return err
}

// {{end}}
// {{define "6dee71c7-41e8-4b3a-8a62-f92d1444bd2e"}}
func (q *Queue) Dequeue() (*client.ObjectRef, error) {
	result, _, err := q.conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
		queue, err := txn.GetObject(q.objRef)
		if err != nil {
			return nil, err
		}
		queueReferences, err := queue.References()
		if err != nil {
			return nil, err
		}
		if len(queueReferences) == 0 {
			// queue is completely empty; Let's wait until it's not!
			return client.Retry, nil

		} else {
			headCell := queueReferences[queueHead]
			headCellReferences, err := headCell.References()
			if err != nil {
				return nil, err
			}
			item := headCellReferences[cellValue]
			if len(headCellReferences) == 1 {
				// there's only one item in the queue and we've just
				// consumed it, so remove all references from the queue
				return item, queue.Set([]byte{})

			} else {
				// the queue head should point to the next cell. The queue
				// tail doesn't change.
				queueReferences[queueHead] = headCellReferences[cellNext]
				return item, queue.Set([]byte{}, queueReferences...)
			}
		}
	})
	if err != nil {
		return nil, err
	}
	itemRef := result.(client.ObjectRef)
	return &itemRef, nil
}

// {{end}}
// {{define "f331337e-8789-4c96-8874-7788ae190c0c"}}
func (q *Queue) Clone(conn *client.Connection) *Queue {
	return &Queue{
		conn:   conn,
		objRef: q.objRef,
	}
}

// {{end}}
