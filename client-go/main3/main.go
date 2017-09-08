// {{define "8b4ed8a8-d495-45c7-9f93-ecf32289acc0"}}
package main

import (
	"encoding/binary"
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
		value := make([]byte, 8)
		binary.LittleEndian.PutUint64(value, 42)
		err = txn.Write(*rootObj, value)
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}
		return "success!", nil
	})
	fmt.Println(result, err)

	result, err = conn.Transact(func(txn *client.Transaction) (interface{}, error) {
		rootObj := txn.Root("myRoot1")
		if rootObj == nil {
			return nil, errors.New("No root 'myRoot1' found")
		}
		value, _, err := txn.Read(*rootObj)
		if err != nil || txn.RestartNeeded() {
			return nil, err
		}
		return fmt.Sprintf("Found value: %v", binary.LittleEndian.Uint64(value)), nil
	})
	fmt.Println(result, err)
}

// {{end}}
