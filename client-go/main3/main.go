// {{define "8b4ed8a8-d495-45c7-9f93-ecf32289acc0"}}
package main

import "goshawkdb.io/client"
import "encoding/binary"
import "fmt"

const (
	clusterCertPEM      = `...`
	clientCertAndKeyPEM = `...`
)

func main() {
	conn, err := client.NewConnection("hostname:7894", []byte(clientCertAndKeyPEM), []byte(clusterCertPEM))
	if err != nil {
		return
	}
	defer conn.Shutdown()

	result, _, err := conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
		rootObjs, err := txn.GetRootObjects()
		if err != nil {
			return nil, err
		}
		rootObj := rootObjs["myRoot1"]
		value := make([]byte, 8)
		binary.LittleEndian.PutUint64(value, 42)
		err = rootObj.Set(value)
		if err != nil {
			return nil, err
		}
		return "success!", nil
	})
	fmt.Println(result, err)

	result, _, err = conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
		rootObjs, err := txn.GetRootObjects()
		if err != nil {
			return nil, err
		}
		rootObj := rootObjs["myRoot1"]
		value, err := rootObj.Value()
		if err != nil {
			return nil, err
		}
		return fmt.Sprintf("Found value: %v", binary.LittleEndian.Uint64(value)), nil
	})
	fmt.Println(result, err)
}

// {{end}}
