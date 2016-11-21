// {{define "2e0a5bb6-70cf-4557-ba82-89347243c938"}}
package bank

import (
	"encoding/json"
	"fmt"
	"goshawkdb.io/client"
	"time"
)

type Bank struct {
	conn   *client.Connection
	objRef client.ObjectRef
}

func CreateBank(conn *client.Connection) (*Bank, error) {
	result, _, err := conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
		roots, err := txn.GetRootObjects()
		if err != nil {
			return nil, err
		}
		rootObj := roots["myRoot1"]
		return rootObj, rootObj.Set([]byte{})
	})
	if err != nil {
		return nil, err
	}
	return &Bank{
		conn:   conn,
		objRef: result.(client.ObjectRef),
	}, nil
}

// {{end}}
// {{define "ea865ae7-bffe-4be4-a2f8-3193f8664d33"}}
type Account struct {
	Bank   *Bank
	objRef client.ObjectRef
	*account
}

type account struct {
	Name          string
	AccountNumber uint
	Balance       int
}

func (b *Bank) AddAccount(name string) (*Account, error) {
	acc := &account{
		Name:    name,
		Balance: 0,
	}
	accObjRef, _, err := b.conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
		bankObj, err := txn.GetObject(b.objRef)
		if err != nil {
			return nil, err
		}
		accounts, err := bankObj.References()
		if err != nil {
			return nil, err
		}
		acc.AccountNumber = uint(len(accounts))
		accValue, err := json.Marshal(acc)
		if err != nil {
			return nil, err
		}
		accObjRef, err := txn.CreateObject(accValue)
		if err != nil {
			return nil, err
		}
		return accObjRef, bankObj.Set([]byte{}, append(accounts, accObjRef)...)
	})
	if err != nil {
		return nil, err
	}
	return &Account{
		Bank:    b,
		objRef:  accObjRef.(client.ObjectRef),
		account: acc,
	}, nil
}

// {{end}}
// {{define "841d6b23-413d-454b-80fa-790893977b38"}}
func (b *Bank) GetAccount(accNum uint) (*Account, error) {
	acc := &account{}
	result, _, err := b.conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
		bankObj, err := txn.GetObject(b.objRef)
		if err != nil {
			return nil, err
		}
		if accounts, err := bankObj.References(); err != nil {
			return nil, err
		} else if accNum < uint(len(accounts)) {
			accObjRef := accounts[accNum]
			accValue, err := accObjRef.Value()
			if err != nil {
				return nil, err
			}
			return accObjRef, json.Unmarshal(accValue, acc)
		} else {
			return nil, fmt.Errorf("Unknown account number: %v", accNum)
		}
	})
	if err != nil {
		return nil, err
	}
	return &Account{
		Bank:    b,
		objRef:  result.(client.ObjectRef),
		account: acc,
	}, nil
}

// {{end}}
// {{define "00669845-8ab8-4382-ba75-b23deca839e5"}}
type transfer struct {
	Time   time.Time
	Amount int
}

type Transfer struct {
	objRef client.ObjectRef
	From   *Account
	To     *Account
	*transfer
}

func (dest *Account) TransferFrom(src *Account, amount int) (*Transfer, error) {
	if src != nil {
		if src.AccountNumber == dest.AccountNumber {
			return nil, fmt.Errorf("Transfer is from and to the same account: %v", src.AccountNumber)
		}
		if !src.Bank.objRef.ReferencesSameAs(dest.Bank.objRef) {
			return nil, fmt.Errorf("Transfer is not within the same bank!")
		}
	}
	t := &transfer{Amount: amount}
	result, _, err := dest.Bank.conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
		t.Time = time.Now() // first, let's create the transfer object
		transferValue, err := json.Marshal(t)
		if err != nil {
			return nil, err
		}
		destAccObjRef, err := txn.GetObject(dest.objRef)
		if err != nil {
			return nil, err
		}

		// the transfer has at least a reference to the destination account
		transferReferences := []client.ObjectRef{destAccObjRef}
		transferObj, err := txn.CreateObject(transferValue, transferReferences...)
		if err != nil {
			return nil, err
		}

		destReferences, err := destAccObjRef.References() // now we must update the dest account
		if err != nil {
			return nil, err
		}
		// append our transfer to the list of destination transfers
		destReferences = append(destReferences, transferObj)
		destValue, err := destAccObjRef.Value()
		if err != nil {
			return nil, err
		}
		destAcc := &account{}
		if err = json.Unmarshal(destValue, destAcc); err != nil {
			return nil, err
		}
		destAcc.Balance += t.Amount // destination is credited the transfer amount
		destValue, err = json.Marshal(destAcc)
		if err != nil {
			return nil, err
		}
		if err = destAccObjRef.Set(destValue, destReferences...); err != nil {
			return nil, err
		}

		if src != nil { // if we have a src, we must update the source account
			srcAccObjRef, err := txn.GetObject(src.objRef)
			if err != nil {
				return nil, err
			}
			srcReferences, err := srcAccObjRef.References()
			if err != nil {
				return nil, err
			}
			// append our transfer to the list of source transfers
			srcReferences = append(srcReferences, transferObj)
			srcValue, err := srcAccObjRef.Value()
			if err != nil {
				return nil, err
			}
			srcAcc := &account{}
			if err = json.Unmarshal(srcValue, srcAcc); err != nil {
				return nil, err
			}
			srcAcc.Balance -= t.Amount // source is debited the transfer amount
			if srcAcc.Balance < 0 {
				// returning an error will abort the entire transaction.
				return nil, fmt.Errorf("Account %v has insufficient funds.", src.AccountNumber)
			}
			srcValue, err = json.Marshal(srcAcc)
			if err != nil {
				return nil, err
			}
			if err = srcAccObjRef.Set(srcValue, srcReferences...); err != nil {
				return nil, err
			}
			// there is a source so add a ref from the transfer to the source account
			transferReferences = append(transferReferences, srcAccObjRef)
			if err = transferObj.Set(transferValue, transferReferences...); err != nil {
				return nil, err
			}
		}

		return transferObj, nil
	})

	if err != nil {
		return nil, err
	}
	return &Transfer{
		objRef:   result.(client.ObjectRef),
		From:     src,
		To:       dest,
		transfer: t,
	}, nil
}

// {{end}}
// {{define "9ef6324a-d118-4882-ae2e-d7ac3ce55270"}}
func (b *Bank) CashDeposit(accNum uint, amount int) (*Transfer, error) {
	result, _, err := b.conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
		account, err := b.GetAccount(accNum)
		if err != nil {
			return nil, err
		}
		return account.TransferFrom(nil, amount)
	})
	if err != nil {
		return nil, err
	}
	return result.(*Transfer), nil
}

func (b *Bank) TransferBetweenAccounts(srcAccNum, destAccNum uint, amount int) (*Transfer, error) {
	result, _, err := b.conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
		srcAccount, err := b.GetAccount(srcAccNum)
		if err != nil {
			return nil, err
		}
		destAccount, err := b.GetAccount(destAccNum)
		if err != nil {
			return nil, err
		}
		return destAccount.TransferFrom(srcAccount, amount)
	})
	if err != nil {
		return nil, err
	}
	return result.(*Transfer), nil
}

// {{end}}
