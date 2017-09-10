// {{define "2e0a5bb6-70cf-4557-ba82-89347243c938"}}
package bank

import (
	"encoding/json"
	"errors"
	"fmt"
	"goshawkdb.io/client"
	"time"
)

type Bank struct {
	objRef client.RefCap
}

func CreateBank(txr client.Transactor) (*Bank, error) {
	result, err := txr.Transact(func(txn *client.Transaction) (interface{}, error) {
		rootObj, found := txn.Root("myRoot1")
		if !found {
			return nil, errors.New("No root 'myRoot1' found")
		}
		return rootObj, txn.Write(rootObj, []byte{})
	})
	if err != nil {
		return nil, err
	}
	return &Bank{
		objRef: result.(client.RefCap),
	}, nil
}

// {{end}}
// {{define "ea865ae7-bffe-4be4-a2f8-3193f8664d33"}}
type Account struct {
	Bank   *Bank
	objRef client.RefCap
	*account
}

type account struct {
	Name          string
	AccountNumber uint
	Balance       int
}

func (b *Bank) AddAccount(txr client.Transactor, name string) (*Account, error) {
	acc := &account{
		Name:    name,
		Balance: 0,
	}
	accObjRef, err := txr.Transact(func(txn *client.Transaction) (interface{}, error) {
		if _, accounts, err := txn.Read(b.objRef); err != nil || txn.RestartNeeded() {
			return nil, err
		} else {
			acc.AccountNumber = uint(len(accounts))
			if accValue, err := json.Marshal(acc); err != nil {
				return nil, err
			} else if accObjRef, err := txn.Create(accValue); err != nil || txn.RestartNeeded() {
				return nil, err
			} else {
				return accObjRef, txn.Write(b.objRef, []byte{}, append(accounts, accObjRef)...)
			}
		}
	})
	if err != nil {
		return nil, err
	}
	return &Account{
		Bank:    b,
		objRef:  accObjRef.(client.RefCap),
		account: acc,
	}, nil
}

// {{end}}
// {{define "841d6b23-413d-454b-80fa-790893977b38"}}
func (b *Bank) GetAccount(txr client.Transactor, accNum uint) (*Account, error) {
	acc := &account{}
	result, err := txr.Transact(func(txn *client.Transaction) (interface{}, error) {
		if _, accounts, err := txn.Read(b.objRef); err != nil || txn.RestartNeeded() {
			return nil, err
		} else if accNum < uint(len(accounts)) {
			accObjRef := accounts[accNum]
			if accValue, _, err := txn.Read(accObjRef); err != nil || txn.RestartNeeded() {
				return nil, err
			} else {
				return accObjRef, json.Unmarshal(accValue, acc)
			}
		} else {
			return nil, fmt.Errorf("Unknown account number: %v", accNum)
		}
	})
	if err != nil {
		return nil, err
	}
	return &Account{
		Bank:    b,
		objRef:  result.(client.RefCap),
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
	objRef client.RefCap
	From   *Account
	To     *Account
	*transfer
}

func (dest *Account) TransferFrom(txr client.Transactor, src *Account, amount int) (*Transfer, error) {
	if src != nil {
		if src.AccountNumber == dest.AccountNumber {
			return nil, fmt.Errorf("Transfer is from and to the same account: %v", src.AccountNumber)
		}
		if !src.Bank.objRef.SameReferent(dest.Bank.objRef) {
			return nil, fmt.Errorf("Transfer is not within the same bank!")
		}
	}
	t := &transfer{Amount: amount}
	result, err := txr.Transact(func(txn *client.Transaction) (interface{}, error) {
		t.Time = time.Now() // first, let's create the transfer object
		transferValue, err := json.Marshal(t)
		if err != nil {
			return nil, err
		}

		// the transfer has at least a reference to the destination account
		transferReferences := []client.RefCap{dest.objRef}
		transferObj, err := txn.Create(transferValue, transferReferences...)
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}

		destValue, destReferences, err := txn.Read(dest.objRef) // now we must update the dest account
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}
		// append our transfer to the list of destination transfers
		destReferences = append(destReferences, transferObj)
		destAcc := &account{}
		if err = json.Unmarshal(destValue, destAcc); err != nil {
			return nil, err
		}
		destAcc.Balance += t.Amount // destination is credited the transfer amount
		destValue, err = json.Marshal(destAcc)
		if err != nil {
			return nil, err
		}
		if err = txn.Write(dest.objRef, destValue, destReferences...); err != nil || txn.RestartNeeded() {
			return nil, err
		}

		if src != nil { // if we have a src, we must update the source account
			srcValue, srcReferences, err := txn.Read(src.objRef)
			if err != nil || txn.RestartNeeded() {
				return nil, err
			}
			// append our transfer to the list of source transfers
			srcReferences = append(srcReferences, transferObj)
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
			if err = txn.Write(src.objRef, srcValue, srcReferences...); err != nil || txn.RestartNeeded() {
				return nil, err
			}
			// there is a source so add a ref from the transfer to the source account
			transferReferences = append(transferReferences, src.objRef)
			if err = txn.Write(transferObj, transferValue, transferReferences...); err != nil || txn.RestartNeeded() {
				return nil, err
			}
		}

		return transferObj, nil
	})

	if err != nil {
		return nil, err
	}
	return &Transfer{
		objRef:   result.(client.RefCap),
		From:     src,
		To:       dest,
		transfer: t,
	}, nil
}

// {{end}}
// {{define "9ef6324a-d118-4882-ae2e-d7ac3ce55270"}}
func (b *Bank) CashDeposit(txr client.Transactor, accNum uint, amount int) (*Transfer, error) {
	result, err := txr.Transact(func(txn *client.Transaction) (interface{}, error) {
		destAccount, err := b.GetAccount(txn, accNum)
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}
		return destAccount.TransferFrom(txn, nil, amount)
	})
	if err != nil {
		return nil, err
	}
	return result.(*Transfer), nil
}

func (b *Bank) TransferBetweenAccounts(txr client.Transactor, srcAccNum, destAccNum uint, amount int) (*Transfer, error) {
	result, err := txr.Transact(func(txn *client.Transaction) (interface{}, error) {
		srcAccount, err := b.GetAccount(txn, srcAccNum)
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}
		destAccount, err := b.GetAccount(txn, destAccNum)
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}
		return destAccount.TransferFrom(txn, srcAccount, amount)
	})
	if err != nil {
		return nil, err
	}
	return result.(*Transfer), nil
}

// {{end}}
