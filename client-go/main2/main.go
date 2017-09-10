// {{define "399af6d4-7892-4393-bb12-ec382849fa75"}}
package main

import (
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
		return "hello", nil
	})
	fmt.Println(result, err)
}

// {{end}}
