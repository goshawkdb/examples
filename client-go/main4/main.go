// {{define "c78fe028-b193-4755-8a23-e8b31b458ff8"}}
package main

import (
	"errors"
	"fmt"
	"goshawkdb.io/client"
)

const (
	clusterCertPEM      = `...`
	clientCertAndKeyPEM = `...`
)

func main() {
	conn, err := client.NewConnection("hostname:7894", []byte(clientCertAndKeyPEM), []byte(clusterCertPEM), nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.ShutdownSync()

	result, err := conn.Transact(func(txn *client.Transaction) (interface{}, error) {
		rootObj := txn.Root("myRoot1")
		if rootObj == nil {
			return nil, errors.New("No root 'myRoot1' found")
		}
		myObj, err := txn.Create([]byte("a new value for a new object"))
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}
		err = txn.Write(*rootObj, nil, myObj) // Root now has no value and one reference
		return "success!", err
	})
	fmt.Println(result, err)

	result, err = conn.Transact(func(txn *client.Transaction) (interface{}, error) {
		rootObj := txn.Root("myRoot1")
		if rootObj == nil {
			return nil, errors.New("No root 'myRoot1' found")
		}
		_, refs, err := txn.Read(*rootObj)
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}
		myObj := refs[0]
		value, _, err := txn.Read(myObj)
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}
		return fmt.Sprintf("Found value: %s", value), nil
	})
	fmt.Println(result, err)
}

// {{end}}
