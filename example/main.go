package main

import (
	"fmt"
	"log"
	"time"

	"github.com/eddielth/modbus"
)

func main() {
	// Example 1: Basic client usage
	basicExample()

	// Example 2: Batch operations
	batchExample()

	// Example 3: Connection pool for high-performance scenarios
	poolExample()
}

// basicExample demonstrates basic Modbus operations
func basicExample() {
	fmt.Println("=== Basic Modbus Operations ===")

	// Create a new Modbus client
	config := modbus.ClientConfig{
		Address: "192.168.1.100:502", // Replace with your device's IP
		Timeout: 5 * time.Second,
	}

	client, err := modbus.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	slaveID := byte(1) // Replace with your device's slave ID

	// Read coils
	fmt.Println("Reading coils...")
	coils, err := client.ReadCoils(slaveID, 0, 10)
	if err != nil {
		log.Printf("Failed to read coils: %v", err)
	} else {
		fmt.Printf("Coils 0-9: %v\n", coils)
	}

	// Read holding registers
	fmt.Println("Reading holding registers...")
	registers, err := client.ReadHoldingRegisters(slaveID, 0, 5)
	if err != nil {
		log.Printf("Failed to read holding registers: %v", err)
	} else {
		fmt.Printf("Holding registers 0-4: %v\n", registers)
	}

	// Write single coil
	fmt.Println("Writing single coil...")
	err = client.WriteSingleCoil(slaveID, 0, true)
	if err != nil {
		log.Printf("Failed to write coil: %v", err)
	} else {
		fmt.Println("Successfully wrote coil 0 = true")
	}

	// Write single register
	fmt.Println("Writing single register...")
	err = client.WriteSingleRegister(slaveID, 0, 1234)
	if err != nil {
		log.Printf("Failed to write register: %v", err)
	} else {
		fmt.Println("Successfully wrote register 0 = 1234")
	}

	// Write multiple coils
	fmt.Println("Writing multiple coils...")
	coilValues := []bool{true, false, true, false, true}
	err = client.WriteMultipleCoils(slaveID, 10, coilValues)
	if err != nil {
		log.Printf("Failed to write multiple coils: %v", err)
	} else {
		fmt.Printf("Successfully wrote coils 10-14: %v\n", coilValues)
	}

	// Write multiple registers
	fmt.Println("Writing multiple registers...")
	regValues := []uint16{100, 200, 300, 400, 500}
	err = client.WriteMultipleRegisters(slaveID, 10, regValues)
	if err != nil {
		log.Printf("Failed to write multiple registers: %v", err)
	} else {
		fmt.Printf("Successfully wrote registers 10-14: %v\n", regValues)
	}
}

// batchExample demonstrates batch operations for better performance
func batchExample() {
	fmt.Println("\n=== Batch Operations ===")

	config := modbus.ClientConfig{
		Address: "192.168.1.100:502",
		Timeout: 5 * time.Second,
	}

	client, err := modbus.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	slaveID := byte(1)

	// Define batch operations
	operations := []modbus.BatchOperation{
		{
			Operation: "read_coils",
			SlaveID:   slaveID,
			Address:   0,
			Quantity:  16,
		},
		{
			Operation: "read_holding",
			SlaveID:   slaveID,
			Address:   0,
			Quantity:  10,
		},
		{
			Operation: "write_coils",
			SlaveID:   slaveID,
			Address:   20,
			Values:    []bool{true, false, true, true, false},
		},
		{
			Operation: "write_registers",
			SlaveID:   slaveID,
			Address:   20,
			Values:    []uint16{1000, 2000, 3000},
		},
	}

	// Execute batch operations
	fmt.Println("Executing batch operations...")
	start := time.Now()
	results := client.ExecuteBatch(operations)
	duration := time.Since(start)

	fmt.Printf("Batch operations completed in %v\n", duration)

	// Process results
	for i, result := range results {
		fmt.Printf("Operation %d (%s): ", i+1, result.Operation)
		if result.Error != nil {
			fmt.Printf("Error: %v\n", result.Error)
		} else {
			switch result.Operation {
			case "read_coils":
				coils := result.Values.([]bool)
				fmt.Printf("Success, read %d coils: %v\n", len(coils), coils)
			case "read_holding", "read_input":
				registers := result.Values.([]uint16)
				fmt.Printf("Success, read %d registers: %v\n", len(registers), registers)
			default:
				fmt.Println("Success")
			}
		}
	}
}

// poolExample demonstrates connection pooling for high-performance scenarios
func poolExample() {
	fmt.Println("\n=== Connection Pool Example ===")

	// Create connection pool
	pool, err := modbus.NewConnectionPool("192.168.1.100:502", 5, 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	slaveID := byte(1)

	// Simulate concurrent operations
	fmt.Println("Performing concurrent operations...")
	start := time.Now()

	// Use goroutines to simulate concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Get connection from pool
			client, err := pool.Get()
			if err != nil {
				log.Printf("Worker %d: Failed to get connection: %v", id, err)
				return
			}
			defer pool.Put(client) // Return connection to pool

			// Perform operation
			registers, err := client.ReadHoldingRegisters(slaveID, uint16(id), 1)
			if err != nil {
				log.Printf("Worker %d: Failed to read register: %v", id, err)
				return
			}

			fmt.Printf("Worker %d: Read register %d = %v\n", id, id, registers)
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	duration := time.Since(start)
	fmt.Printf("Concurrent operations completed in %v\n", duration)
}

// advancedExample demonstrates advanced features like float handling
func advancedExample() {
	fmt.Println("\n=== Advanced Features ===")

	config := modbus.ClientConfig{
		Address: "192.168.1.100:502",
		Timeout: 5 * time.Second,
	}

	client, err := modbus.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	slaveID := byte(1)

	// Read float32 value (stored in 2 consecutive registers)
	fmt.Println("Reading float32 value...")
	floatValue, err := client.ReadFloat32(slaveID, 100, "big")
	if err != nil {
		log.Printf("Failed to read float32: %v", err)
	} else {
		fmt.Printf("Float32 value at register 100-101: %f\n", floatValue)
	}

	// Write float32 value
	fmt.Println("Writing float32 value...")
	err = client.WriteFloat32(slaveID, 102, 3.14159, "big")
	if err != nil {
		log.Printf("Failed to write float32: %v", err)
	} else {
		fmt.Println("Successfully wrote float32 value 3.14159 to register 102-103")
	}

	// Error handling example
	fmt.Println("Testing error handling...")
	_, err = client.ReadHoldingRegisters(slaveID, 65535, 1)
	if err != nil {
		switch e := err.(type) {
		case *modbus.ModbusError:
			fmt.Printf("Modbus exception: Function=0x%02X, Exception=0x%02X\n",
				e.FunctionCode, e.ExceptionCode)
		default:
			fmt.Printf("Other error: %v\n", err)
		}
	}
}
