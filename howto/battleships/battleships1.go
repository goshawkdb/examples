// {{define "9f869cf9-a5d5-4f78-88b8-fae101be9e33"}}
package battleships

import (
	"fmt"
	"goshawkdb.io/client"
)

type Game struct {
	objId   client.ObjectRef
	playerA client.ObjectRef
	playerB client.ObjectRef
}

type Player struct {
	conn          *client.Connection
	board         client.ObjectRef
	boats         client.ObjectRef
	opponentBoard client.ObjectRef
}

// {{end}}
// {{define "1d13849b-998c-40eb-9f9f-47d18b7f1090"}}
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

// {{end}}
// {{define "ed621b49-1b94-480a-a1ab-bdc361522bd8"}}
func (p *Player) Shoot(x, y uint) error {
	cellIndex := y*boardXDim + x
	_, _, err := p.conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
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
			return nil, cell.Set([]byte{0}) // shoot it!
		}
	})
	return err
}

// {{end}}
// {{define "1b2e32ee-1acd-4c6b-a6e6-a2159588baf0"}}
func Referee(conn *client.Connection, playerABoats, playerBBoats []client.ObjectRef) error {
	for len(playerABoats) != 0 && len(playerBBoats) != 0 {
		var boats []client.ObjectRef
		result, _, err := conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
			for i, cells := range [][]client.ObjectRef{playerABoats, playerBBoats} {
				for j, cell := range cells {
					if cellValue, err := cell.Value(); err != nil {
						return nil, err
					} else if len(cellValue) != 0 {
						// this cell has been shot!
						boats = make([]client.ObjectRef, len(cells)-1)
						copy(boats, cells[:j])
						copy(boats[j:], cells[j+1:])
						return i, nil
					}
				}
			}
			// all the cells we've looked through are unshot.
			return client.Retry, nil
		})
		if err != nil {
			return err
		}
		player := result.(int)
		if player == 0 {
			playerABoats = boats
			fmt.Println("Player B HIT a player A boat!")
		} else {
			playerBBoats = boats
			fmt.Println("Player A HIT a player B boat!")
		}
	}
	if len(playerABoats) == 0 {
		fmt.Println("Player B wins!")
	} else {
		fmt.Println("Player A wins!")
	}
	return nil
}

// {{end}}
