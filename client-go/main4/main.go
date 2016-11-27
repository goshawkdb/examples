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
	conn, err := client.NewConnection("hostname:7894", []byte(clientCertAndKeyPEM), []byte(clusterCertPEM))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Shutdown()

	result, _, err := conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
		rootObjs, err := txn.GetRootObjects()
		if err != nil {
			return nil, err
		}
		rootObj, found := rootObjs["myRoot1"]
		if !found {
			return nil, errors.New("No root 'myRoot1' found")
		}
		myObj, err := txn.CreateObject([]byte("a new value for a new object"))
		if err != nil {
			return nil, err
		}
		rootObj.Set([]byte{}, myObj) // Root now has no value and one reference
		return "success!", nil
	})
	fmt.Println(result, err)

	result, _, err = conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
		rootObjs, err := txn.GetRootObjects()
		if err != nil {
			return nil, err
		}
		rootObj, found := rootObjs["myRoot1"]
		if !found {
			return nil, errors.New("No root 'myRoot1' found")
		}
		refs, err := rootObj.References()
		if err != nil {
			return nil, err
		}
		myObj := refs[0]
		value, err := myObj.Value()
		if err != nil {
			return nil, err
		}
		return fmt.Sprintf("Found value: %s", value), nil
	})
	fmt.Println(result, err)
}

// {{end}}
