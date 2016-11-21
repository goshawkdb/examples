package battleships

import (
	"fmt"
	"goshawkdb.io/client"
	"goshawkdb.io/examples/howto/queue"
)

type Game struct {
	objId   client.ObjectRef
	playerA client.ObjectRef
	playerB client.ObjectRef
}

const (
	boardXDim = 10
	boardYDim = 10
)

func (p *Player) PlaceBoat(x, y, boatLength uint, vertical bool) error {
	switch {
	case vertical && y+boatLength >= boardYDim:
		return fmt.Errorf("Illegal position for boat")
	case !vertical && x+boatLength >= boardXDim:
		return fmt.Errorf("Illegal position for boat")
	}
	cellIndices := make([]uint, boatLength)
	if vertical {
		for idx := range cellIndices {
			cellIndices[idx] = y*boardXDim + x + uint(idx)*boardXDim
		}
	} else {
		for idx := range cellIndices {
			cellIndices[idx] = y*boardXDim + x + uint(idx)
		}
	}
	_, _, err := p.conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
		boardObj, err := txn.GetObject(p.board)
		if err != nil {
			return nil, err
		}
		boardCells, err := boardObj.References()
		if err != nil {
			return nil, err
		}
		boatsObj, err := txn.GetObject(p.boats)
		if err != nil {
			return nil, err
		}
		boatsObjReferences, err := boatsObj.References()
		if err != nil {
			return nil, err
		}
		for _, index := range cellIndices {
			cell := boardCells[index]
			boatsObjReferences = append(boatsObjReferences, cell)
		}
		boatsObj.Set([]byte{}, boatsObjReferences...)
		return nil, nil
	})
	return err
}

// {{define "bfd6d594-76a4-4a19-8149-667a93f645d9"}}
type Player struct {
	conn          *client.Connection
	board         client.ObjectRef
	boats         client.ObjectRef
	opponentBoard client.ObjectRef
	shotQueue     *queue.Queue
}

func (p *Player) Shoot(x, y uint) (bool, error) {
	cellIndex := y*boardXDim + x
	result, _, err := p.conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
		boardObj, err := txn.GetObject(p.opponentBoard)
		if err != nil {
			return nil, err
		}
		boardCells, err := boardObj.References()
		if err != nil {
			return nil, err
		}
		cell := boardCells[cellIndex]
		if cellValue, err := cell.Value(); err != nil {
			return nil, err
		} else if len(cellValue) != 0 {
			return nil, fmt.Errorf("Illegal shot: already been shot!")
		} else {
			return cell, nil
		}
	})
	if err != nil {
		return false, err
	}
	cell := result.(client.ObjectRef)
	if err = p.shotQueue.Append(cell); err != nil {
		return false, err
	}
	result, _, err = p.conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
		cell, err := txn.GetObject(cell)
		if err != nil {
			return nil, err
		}
		if cellValue, err := cell.Value(); err != nil {
			return nil, err
		} else if len(cellValue) != 0 {
			// shot has been carried out. Referee would have written a 1 if we hit.
			return cellValue[0] == 1, nil
		} else {
			// shot has not happend yet
			return client.Retry, nil
		}
	})
	if err != nil {
		return false, err
	}
	return result.(bool), nil
}

// {{end}}
// {{define "6aacfa5f-015a-4196-b970-6d115b44b37a"}}
type Referee struct {
	conn             *client.Connection
	playerABoats     []client.ObjectRef
	playerBBoats     []client.ObjectRef
	playerAShotQueue *queue.Queue
	playerBShotQueue *queue.Queue
}

func (r *Referee) Run() error {
	turnQ, notTurnQ := r.playerAShotQueue, r.playerBShotQueue
	turnBoats, notTurnBoats := r.playerBBoats, r.playerABoats
	for len(turnBoats) != 0 && len(notTurnBoats) != 0 {
		result, _, err := r.conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
			cell, err := turnQ.Dequeue()
			if err != nil {
				return nil, err
			}
			shotOutcome := []byte{0}
			for i, boatCell := range turnBoats {
				if cell.ReferencesSameAs(boatCell) {
					shotOutcome[0] = 1 // hit!
					return i, cell.Set(shotOutcome)
				}
			}
			return nil, cell.Set(shotOutcome) // miss
		})
		if err != nil {
			return err
		}
		if result != nil { // remove hit boat cell
			i := result.(int)
			turnBoats = append(turnBoats[:i], turnBoats[i+1:]...)
		}
		// swap turns
		turnBoats, notTurnBoats = notTurnBoats, turnBoats
		turnQ, notTurnQ = notTurnQ, turnQ
	}
	return nil
}

// {{end}}
