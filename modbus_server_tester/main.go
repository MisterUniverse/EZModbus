// client.go - FINAL CORRECTED VERSION
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/simonvetter/modbus"
)

var stats struct {
	successes atomic.Uint64
	failures  atomic.Uint64
}

func main() {
	serverURL := flag.String("url", "tcp://localhost:1502", "Modbus server URL (e.g., tcp://127.0.0.1:1502)")
	unitID := flag.Uint("unitID", 1, "The correct Modbus Unit ID of the server")
	numClients := flag.Int("clients", 5, "Number of concurrent clients to simulate")
	runDuration := flag.Duration("duration", 30*time.Second, "How long to run the test for")
	requestsPerSec := flag.Int("rate", 10, "Requests per second for each client")
	counterAddr := flag.Uint("counterAddr", 102, "Address of the server's auto-incrementing counter")
	flag.Parse()

	log.Printf("Starting Modbus stress test...")
	log.Printf("Target: %s, UnitID: %d, Concurrent Clients: %d", *serverURL, *unitID, *numClients)
	log.Printf("Test Duration: %v, Request Rate: %d/sec per client", *runDuration, *requestsPerSec)
	log.Println("--------------------------------------------------")

	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), *runDuration)
	defer cancel()

	for i := 0; i < *numClients; i++ {
		wg.Add(1)
		go runTestClient(ctx, &wg, i+1, *serverURL, uint8(*unitID), *requestsPerSec, uint16(*counterAddr))
	}

	wg.Wait()

	log.Println("--------------------------------------------------")
	log.Printf("Test finished. Total Successes: %d, Total Failures: %d\n", stats.successes.Load(), stats.failures.Load())
}

func runTestClient(ctx context.Context, wg *sync.WaitGroup, clientID int, url string, unitID uint8, rate int, counterAddr uint16) {
	defer wg.Done()
	l := log.New(os.Stdout, fmt.Sprintf("[Client %d] ", clientID), log.Ltime)

	client, err := modbus.NewClient(&modbus.ClientConfiguration{
		URL:     url,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		l.Printf("ERROR: Failed to create client: %v", err)
		stats.failures.Add(1)
		return
	}
	if err = client.Open(); err != nil {
		l.Printf("ERROR: Failed to open connection: %v", err)
		stats.failures.Add(1)
		return
	}
	defer client.Close()
	l.Println("Connected successfully.")

	ticker := time.NewTicker(time.Second / time.Duration(rate))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			l.Println("Test duration ended. Disconnecting.")
			return
		case <-ticker.C:
			runTestSequence(l, client, unitID, clientID, counterAddr)
		}
	}
}

// runTestSequence performs a series of validation checks
func runTestSequence(l *log.Logger, client *modbus.ModbusClient, unitID uint8, clientID int, counterAddr uint16) {
	client.SetUnitId(unitID)

	// Test 1: Data Integrity (Write then Read)
	testAddr := uint16(200 + clientID)
	testValue := uint16(1000 + clientID)
	err := client.WriteRegister(testAddr, testValue)
	if err == nil {
		stats.successes.Add(1)
		readVal, err_read := client.ReadRegister(testAddr, modbus.HOLDING_REGISTER)
		if err_read == nil && readVal == testValue {
			stats.successes.Add(1)
		} else {
			l.Printf("FAIL: Data integrity check failed. Wrote %d, but read %d. Error: %v", testValue, readVal, err_read)
			stats.failures.Add(1)
		}
	} else {
		l.Printf("FAIL: Could not write to register %d: %v", testAddr, err)
		stats.failures.Add(1)
	}

	// Test 2: Protected Register
	err = client.WriteRegister(counterAddr, 9999)
	if err == nil {
		stats.successes.Add(1)
		val, err_read := client.ReadRegister(counterAddr, modbus.HOLDING_REGISTER)
		if err_read == nil && val != 9999 {
			stats.successes.Add(1)
		} else {
			l.Printf("FAIL: Protected register test failed. Wrote to counter, but value changed to %d. Error: %v", val, err_read)
			stats.failures.Add(1)
		}
	} else {
		l.Printf("FAIL: Could not write to protected register %d: %v", counterAddr, err)
		stats.failures.Add(1)
	}

	// Test 3: Counter Check
	counter1, err_c1 := client.ReadRegister(counterAddr, modbus.HOLDING_REGISTER)
	time.Sleep(1100 * time.Millisecond)
	counter2, err_c2 := client.ReadRegister(counterAddr, modbus.HOLDING_REGISTER)
	if err_c1 == nil && err_c2 == nil && counter2 > counter1 {
		stats.successes.Add(2)
	} else {
		l.Printf("FAIL: Counter check failed. First read: %d, Second read: %d. Errors: %v, %v", counter1, counter2, err_c1, err_c2)
		stats.failures.Add(1)
	}

	// Test 4: Invalid Unit ID
	client.SetUnitId(99)
	_, err = client.ReadRegister(100, modbus.HOLDING_REGISTER)
	if err != nil {
		stats.successes.Add(1)
	} else {
		l.Printf("FAIL: Invalid Unit ID test failed. Server did not return an error.")
		stats.failures.Add(1)
	}
	client.SetUnitId(unitID)

	// Test 5: Out of Bounds Read
	_, err = client.ReadRegisters(9999, 1, modbus.HOLDING_REGISTER)
	if err != nil {
		stats.successes.Add(1)
	} else {
		l.Printf("FAIL: Out of Bounds test failed. Server did not return an error.")
		stats.failures.Add(1)
	}
}
