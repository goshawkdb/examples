// {{define "4dd45fdf-4778-4418-b44c-ad559fe9296c"}}
package main

import (
	"fmt"
	"goshawkdb.io/client"
	"goshawkdb.io/examples/howto/queue"
	"log"
)

const (
	clusterCertPEM      = `...`
	clientCertAndKeyPEM = `...`
)

func main() {
	consumerCount := 2
	producerCount := 3

	productionLimit := 10

	connections := make([]*client.Connection, consumerCount+producerCount)
	consumers := connections[:consumerCount]
	producers := connections[consumerCount:]

	for i := range connections {
		conn, err := client.NewConnection("localhost", []byte(clientCertAndKeyPEM), []byte(clusterCertPEM))
		if err != nil {
			log.Fatal(err)
		}
		connections[i] = conn
		defer conn.Shutdown()
	}

	q, err := queue.NewQueue(connections[0])
	if err != nil {
		log.Fatal(err)
	}

	for i, conn := range consumers {
		consumer := i
		connection := conn
		go consume(consumer, connection, q)
	}

	for i, conn := range producers {
		producer := i
		connection := conn
		go produce(producer, connection, q, productionLimit)
	}

	// wait forever
	c := make(chan struct{})
	<-c
}

func consume(consumerId int, conn *client.Connection, q *queue.Queue) {
	q = q.Clone(conn)
	for {
		result, _, err := conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
			item, err := q.Dequeue()
			if err != nil {
				return nil, err
			}
			itemValue, err := item.Value()
			if err != nil {
				return nil, err
			}
			return itemValue, nil
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Consumer %v dequeued '%s'\n", consumerId, result)
	}
}

func produce(producerId int, conn *client.Connection, q *queue.Queue, limit int) {
	q = q.Clone(conn)
	for i := 0; i < limit; i++ {
		_, _, err := conn.RunTransaction(func(txn *client.Txn) (interface{}, error) {
			itemValue := []byte(fmt.Sprintf("producer %v item %v", producerId, i))
			item, err := txn.CreateObject(itemValue)
			if err != nil {
				return nil, err
			}
			return nil, q.Append(item)
		})
		if err != nil {
			log.Fatal(err)
		}
	}
}

// {{end}}
