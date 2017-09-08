// {{define "ce212aca-5f6c-4031-8076-f4b3ac2e3ea0"}}
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
	conn1, err := client.NewConnection("hostname:7894", []byte(clientCertAndKeyPEM), []byte(clusterCertPEM), nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn1.ShutdownSync()
	conn2, err := client.NewConnection("hostname:7894", []byte(clientCertAndKeyPEM), []byte(clusterCertPEM), nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	limit := uint64(1000)
	go func() { // producer
		defer conn2.ShutdownSync()
		buf := make([]byte, 8)
		fmt.Println("Producer starting")
		for i := uint64(0); i < limit; i++ {
			_, err := conn2.Transact(func(txn *client.Transaction) (interface{}, error) {
				rootObj := txn.Root("myRoot1")
				if rootObj == nil {
					return nil, errors.New("No root 'myRoot1' found")
				}
				binary.LittleEndian.PutUint64(buf, i)
				return nil, txn.Write(*rootObj, buf)
			})
			if err != nil {
				fmt.Println(err)
				return
			}
		}
		fmt.Println("Producer finished")
	}()

	retrieved := uint64(0)
	for retrieved+1 != limit { // consumer
		result, err := conn1.Transact(func(txn *client.Transaction) (interface{}, error) {
			rootObj := txn.Root("myRoot1")
			if rootObj == nil {
				return nil, errors.New("No root 'myRoot1' found")
			}
			value, _, err := txn.Read(*rootObj)
			if err != nil || txn.RestartNeeded() {
				return nil, err
			}
			if len(value) == 0 {
				// the producer hasn't written anything yet: go to sleep!
				return nil, txn.Retry()
			}
			num := binary.LittleEndian.Uint64(value)
			if num == retrieved { // nothing's changed since we last read the root, go to sleep!
				return nil, txn.Retry()
			} else {
				return num, nil
			}
		})
		if err != nil {
			fmt.Println(err)
			return
		}
		retrieved = result.(uint64)
		fmt.Printf("Consumer retrieved %v\n", retrieved)
	}
}

// {{end}}
