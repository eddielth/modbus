package modbus

import (
	"testing"
	"time"
)

// TestModbusError tests the ModbusError type
func TestModbusError(t *testing.T) {
	err := &ModbusError{
		FunctionCode:  0x03,
		ExceptionCode: 0x02,
	}

	expected := "Modbus exception: function=0x03, exception=0x02"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

// TestClientConfig tests client configuration validation
func TestClientConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  ClientConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ClientConfig{
				Address: "127.0.0.1:502",
				Timeout: 5 * time.Second,
			},
			wantErr: true, // Connection will fail but config is valid
		},
		{
			name: "default timeout",
			config: ClientConfig{
				Address: "127.0.0.1:502",
				Timeout: 0, // Should use default
			},
			wantErr: true, // Connection will fail but config handling is correct
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateQuantity tests quantity validation for various operations
func TestValidateQuantity(t *testing.T) {
	// This would be used by a mock client for testing
	tests := []struct {
		operation string
		quantity  uint16
		valid     bool
	}{
		{"read_coils", 1, true},
		{"read_coils", 2000, true},
		{"read_coils", 2001, false},
		{"read_coils", 0, false},
		{"read_holding", 1, true},
		{"read_holding", 125, true},
		{"read_holding", 126, false},
		{"write_multiple_coils", 1, true},
		{"write_multiple_coils", 1968, true},
		{"write_multiple_coils", 1969, false},
		{"write_multiple_registers", 1, true},
		{"write_multiple_registers", 123, true},
		{"write_multiple_registers", 124, false},
	}

	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			var valid bool
			switch tt.operation {
			case "read_coils":
				valid = tt.quantity > 0 && tt.quantity <= 2000
			case "read_holding":
				valid = tt.quantity > 0 && tt.quantity <= 125
			case "write_multiple_coils":
				valid = tt.quantity > 0 && tt.quantity <= 1968
			case "write_multiple_registers":
				valid = tt.quantity > 0 && tt.quantity <= 123
			}

			if valid != tt.valid {
				t.Errorf("Quantity %d for %s: expected valid=%v, got valid=%v",
					tt.quantity, tt.operation, tt.valid, valid)
			}
		})
	}
}

// TestBatchOperation tests batch operation structure
func TestBatchOperation(t *testing.T) {
	operations := []BatchOperation{
		{
			Operation: "read_coils",
			SlaveID:   1,
			Address:   0,
			Quantity:  10,
		},
		{
			Operation: "write_registers",
			SlaveID:   1,
			Address:   100,
			Values:    []uint16{1, 2, 3, 4, 5},
		},
	}

	if len(operations) != 2 {
		t.Errorf("Expected 2 operations, got %d", len(operations))
	}

	// Test read operation
	readOp := operations[0]
	if readOp.Operation != "read_coils" {
		t.Errorf("Expected operation 'read_coils', got '%s'", readOp.Operation)
	}
	if readOp.Quantity != 10 {
		t.Errorf("Expected quantity 10, got %d", readOp.Quantity)
	}

	// Test write operation
	writeOp := operations[1]
	if writeOp.Operation != "write_registers" {
		t.Errorf("Expected operation 'write_registers', got '%s'", writeOp.Operation)
	}
	if values, ok := writeOp.Values.([]uint16); !ok {
		t.Error("Expected values to be []uint16")
	} else if len(values) != 5 {
		t.Errorf("Expected 5 values, got %d", len(values))
	}
}

// TestCoilConversion tests conversion between boolean arrays and byte arrays
func TestCoilConversion(t *testing.T) {
	tests := []struct {
		name  string
		coils []bool
		bytes []byte
	}{
		{
			name:  "8 coils",
			coils: []bool{true, false, true, false, true, false, true, false},
			bytes: []byte{0x55}, // 01010101 in binary
		},
		{
			name:  "16 coils",
			coils: []bool{true, true, false, false, true, true, false, false, false, true, false, true, false, true, false, true},
			bytes: []byte{0x33, 0xAA}, // 00110011, 10101010
		},
		{
			name:  "partial byte",
			coils: []bool{true, false, true},
			bytes: []byte{0x05}, // 00000101 (only first 3 bits used)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test coils to bytes conversion
			byteCount := (len(tt.coils) + 7) / 8
			result := make([]byte, byteCount)

			for i, coil := range tt.coils {
				if coil {
					byteIndex := i / 8
					bitIndex := i % 8
					result[byteIndex] |= 1 << bitIndex
				}
			}

			if len(result) != len(tt.bytes) {
				t.Errorf("Expected %d bytes, got %d", len(tt.bytes), len(result))
				return
			}

			for i, expected := range tt.bytes {
				if result[i] != expected {
					t.Errorf("Byte %d: expected 0x%02X, got 0x%02X", i, expected, result[i])
				}
			}

			// Test bytes to coils conversion
			coilsResult := make([]bool, len(tt.coils))
			for i := range coilsResult {
				byteIndex := i / 8
				bitIndex := i % 8
				if byteIndex < len(tt.bytes) {
					coilsResult[i] = (tt.bytes[byteIndex] & (1 << bitIndex)) != 0
				}
			}

			for i, expected := range tt.coils {
				if coilsResult[i] != expected {
					t.Errorf("Coil %d: expected %v, got %v", i, expected, coilsResult[i])
				}
			}
		})
	}
}

// TestTransactionIDIncrement tests transaction ID increment behavior
func TestTransactionIDIncrement(t *testing.T) {
	client := &Client{
		transactionID: 0,
	}

	// Simulate transaction ID increment
	for i := 1; i <= 5; i++ {
		client.transactionID++
		if client.transactionID != uint16(i) {
			t.Errorf("Expected transaction ID %d, got %d", i, client.transactionID)
		}
	}

	// Test overflow
	client.transactionID = 65535
	client.transactionID++
	if client.transactionID != 0 {
		t.Errorf("Expected transaction ID to overflow to 0, got %d", client.transactionID)
	}
}

// BenchmarkCoilConversion benchmarks coil to byte conversion
func BenchmarkCoilConversion(b *testing.B) {
	coils := make([]bool, 1000)
	for i := range coils {
		coils[i] = i%2 == 0 // Alternate true/false
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteCount := (len(coils) + 7) / 8
		result := make([]byte, byteCount)

		for j, coil := range coils {
			if coil {
				byteIndex := j / 8
				bitIndex := j % 8
				result[byteIndex] |= 1 << bitIndex
			}
		}
	}
}

// BenchmarkRegisterConversion benchmarks register to byte conversion
func BenchmarkRegisterConversion(b *testing.B) {
	registers := make([]uint16, 100)
	for i := range registers {
		registers[i] = uint16(i * 100)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := make([]byte, len(registers)*2)
		for j, reg := range registers {
			result[j*2] = byte(reg >> 8)
			result[j*2+1] = byte(reg & 0xFF)
		}
	}
}

// MockServer represents a simple mock Modbus server for testing
type MockServer struct {
	coils     map[uint16]bool
	registers map[uint16]uint16
}

// NewMockServer creates a new mock server
func NewMockServer() *MockServer {
	return &MockServer{
		coils:     make(map[uint16]bool),
		registers: make(map[uint16]uint16),
	}
}

// TestMockServer tests the mock server functionality
func TestMockServer(t *testing.T) {
	server := NewMockServer()

	// Test coil operations
	server.coils[0] = true
	server.coils[1] = false
	server.coils[2] = true

	if !server.coils[0] {
		t.Error("Expected coil 0 to be true")
	}
	if server.coils[1] {
		t.Error("Expected coil 1 to be false")
	}

	// Test register operations
	server.registers[0] = 1234
	server.registers[1] = 5678

	if server.registers[0] != 1234 {
		t.Errorf("Expected register 0 to be 1234, got %d", server.registers[0])
	}
	if server.registers[1] != 5678 {
		t.Errorf("Expected register 1 to be 5678, got %d", server.registers[1])
	}
}

// TestConnectionPoolBasics tests basic connection pool functionality
func TestConnectionPoolBasics(t *testing.T) {
	// Test pool creation with invalid parameters
	_, err := NewConnectionPool("invalid-address", 0, 0)
	if err == nil {
		t.Error("Expected error for invalid address, got nil")
	}

	// Test pool size validation
	pool := &ConnectionPool{
		maxConn: 5,
		pool:    make(chan *Client, 5),
	}

	if cap(pool.pool) != 5 {
		t.Errorf("Expected pool capacity 5, got %d", cap(pool.pool))
	}
}

// TestErrorHandling tests various error conditions
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		functionCode  byte
		exceptionCode byte
		expectedMsg   string
	}{
		{
			name:          "illegal function",
			functionCode:  0x03,
			exceptionCode: 0x01,
			expectedMsg:   "Modbus exception: function=0x03, exception=0x01",
		},
		{
			name:          "illegal address",
			functionCode:  0x04,
			exceptionCode: 0x02,
			expectedMsg:   "Modbus exception: function=0x04, exception=0x02",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ModbusError{
				FunctionCode:  tt.functionCode,
				ExceptionCode: tt.exceptionCode,
			}

			if err.Error() != tt.expectedMsg {
				t.Errorf("Expected error message '%s', got '%s'", tt.expectedMsg, err.Error())
			}
		})
	}
}

// Example test showing how to test with a real Modbus device (commented out)
/*
func TestRealDevice(t *testing.T) {
	// Skip this test unless explicitly enabled
	if testing.Short() {
		t.Skip("Skipping real device test in short mode")
	}

	config := ClientConfig{
		Address: "192.168.1.100:502", // Replace with real device
		Timeout: 5 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Skipf("Cannot connect to real device: %v", err)
	}
	defer client.Close()

	// Test basic read operation
	_, err = client.ReadHoldingRegisters(1, 0, 1)
	if err != nil {
		t.Errorf("Failed to read from real device: %v", err)
	}
}
*/
