package battleships

import (
	"fmt"
	"goshawkdb.io/client"
	"goshawkdb.io/examples/howto/queue"
)

type Game struct {
	objId   client.RefCap
	playerA client.RefCap
	playerB client.RefCap
}

const (
	boardXDim = 10
	boardYDim = 10
)

func (p *Player) PlaceBoat(txr client.Transactor, x, y, boatLength uint, vertical bool) error {
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
	_, err := txr.Transact(func(txn *client.Transaction) (interface{}, error) {
		_, boardCells, err := txn.Read(p.board)
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}
		_, boatsObjRefs, err := txn.Read(p.boats)
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}
		for _, index := range cellIndices {
			cell := boardCells[index]
			boatsObjRefs = append(boatsObjRefs, cell)
		}
		return nil, txn.Write(p.boats, []byte{}, boatsObjRefs...)
	})
	return err
}

// {{define "bfd6d594-76a4-4a19-8149-667a93f645d9"}}
type Player struct {
	board         client.RefCap
	boats         client.RefCap
	opponentBoard client.RefCap
	shotQueue     *queue.Queue
}

func (p *Player) Shoot(txr client.Transactor, x, y uint) (bool, error) {
	cellIndex := y*boardXDim + x
	result, err := txr.Transact(func(txn *client.Transaction) (interface{}, error) {
		_, boardCells, err := txn.Read(p.opponentBoard)
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}
		cell := boardCells[cellIndex]
		if cellValue, _, err := txn.Read(cell); err != nil || txn.RestartNeeded() {
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
	cell := result.(client.RefCap)
	if err = p.shotQueue.Append(txr, cell); err != nil {
		return false, err
	}
	result, err = txr.Transact(func(txn *client.Transaction) (interface{}, error) {
		if cellValue, _, err := txn.Read(cell); err != nil || txn.RestartNeeded() {
			return nil, err
		} else if len(cellValue) != 0 {
			// shot has been carried out. Referee would have written a 1 if we hit.
			return cellValue[0] == 1, nil
		} else {
			// shot has not happend yet
			return nil, txn.Retry()
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
	playerABoats     []client.RefCap
	playerBBoats     []client.RefCap
	playerAShotQueue *queue.Queue
	playerBShotQueue *queue.Queue
}

func (r *Referee) Run(txr client.Transactor) error {
	turnQ, notTurnQ := r.playerAShotQueue, r.playerBShotQueue
	turnBoats, notTurnBoats := r.playerBBoats, r.playerABoats
	for len(turnBoats) != 0 && len(notTurnBoats) != 0 {
		result, err := txr.Transact(func(txn *client.Transaction) (interface{}, error) {
			cell, err := turnQ.Dequeue(txn)
			if err != nil || txn.RestartNeeded() {
				return nil, err
			}
			shotOutcome := []byte{0}
			for i, boatCell := range turnBoats {
				if cell.SameReferent(boatCell) {
					shotOutcome[0] = 1 // hit!
					return i, txn.Write(cell, shotOutcome)
				}
			}
			return nil, txn.Write(cell, shotOutcome) // miss
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
