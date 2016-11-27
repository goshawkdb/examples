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
	conn1, err := client.NewConnection("hostname:7894", []byte(clientCertAndKeyPEM), []byte(clusterCertPEM))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn1.Shutdown()
	conn2, err := client.NewConnection("hostname:7894", []byte(clientCertAndKeyPEM), []byte(clusterCertPEM))
	if err != nil {
		fmt.Println(err)
		return
	}

	limit := uint64(1000)
	go func() { // producer
		defer conn2.Shutdown()
		buf := make([]byte, 8)
		fmt.Println("Producer starting")
		for i := uint64(0); i < limit; i++ {
			_, _, err := conn2.RunTransaction(func(txn *client.Txn) (interface{}, error) {
				rootObjs, err := txn.GetRootObjects()
				if err != nil {
					return nil, err
				}
				rootObj, found := rootObjs["myRoot1"]
				if !found {
					return nil, errors.New("No root 'myRoot1' found")
				}
				binary.LittleEndian.PutUint64(buf, i)
				return nil, rootObj.Set(buf)
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
		result, _, err := conn1.RunTransaction(func(txn *client.Txn) (interface{}, error) {
			rootObjs, err := txn.GetRootObjects()
			if err != nil {
				return nil, err
			}
			rootObj, found := rootObjs["myRoot1"]
			if !found {
				return nil, errors.New("No root 'myRoot1' found")
			}
			value, err := rootObj.Value()
			if err != nil {
				return nil, err
			}
			if len(value) == 0 {
				// the producer hasn't written anything yet: go to sleep!
				return client.Retry, nil
			}
			num := binary.LittleEndian.Uint64(value)
			if num == retrieved { // nothing's changed since we last read the root, go to sleep!
				return client.Retry, nil
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
