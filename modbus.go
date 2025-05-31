package modbus

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
	"unsafe"
)

// Function codes for Modbus operations
const (
	FuncCodeReadCoils              = 0x01
	FuncCodeReadDiscreteInputs     = 0x02
	FuncCodeReadHoldingRegisters   = 0x03
	FuncCodeReadInputRegisters     = 0x04
	FuncCodeWriteSingleCoil        = 0x05
	FuncCodeWriteSingleRegister    = 0x06
	FuncCodeWriteMultipleCoils     = 0x0F
	FuncCodeWriteMultipleRegisters = 0x10
)

// Exception codes
const (
	ExceptionIllegalFunction    = 0x01
	ExceptionIllegalDataAddress = 0x02
	ExceptionIllegalDataValue   = 0x03
	ExceptionSlaveDeviceFailure = 0x04
)

// ModbusError represents a Modbus exception
type ModbusError struct {
	FunctionCode  byte
	ExceptionCode byte
}

func (e *ModbusError) Error() string {
	return fmt.Sprintf("Modbus exception: function=0x%02X, exception=0x%02X",
		e.FunctionCode, e.ExceptionCode)
}

// Client represents a Modbus TCP client
type Client struct {
	conn          net.Conn
	timeout       time.Duration
	transactionID uint16
	mutex         sync.Mutex
}

// ClientConfig holds configuration for Modbus client
type ClientConfig struct {
	Address string        // TCP address (e.g., "192.168.1.100:502")
	Timeout time.Duration // Operation timeout
}

// NewClient creates a new Modbus TCP client
func NewClient(config ClientConfig) (*Client, error) {
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}

	conn, err := net.DialTimeout("tcp", config.Address, config.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &Client{
		conn:    conn,
		timeout: config.Timeout,
	}, nil
}

// Close closes the connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// sendRequest sends a Modbus request and returns the response
func (c *Client) sendRequest(slaveID byte, pdu []byte) ([]byte, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Increment transaction ID for each request
	c.transactionID++

	// Build MBAP (Modbus Application Protocol) header
	mbap := make([]byte, 7)
	binary.BigEndian.PutUint16(mbap[0:2], c.transactionID)    // Transaction ID
	binary.BigEndian.PutUint16(mbap[2:4], 0)                  // Protocol ID (0 for Modbus)
	binary.BigEndian.PutUint16(mbap[4:6], uint16(len(pdu)+1)) // Length
	mbap[6] = slaveID                                         // Unit ID

	// Combine MBAP header with PDU
	request := append(mbap, pdu...)

	// Set write timeout
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}

	// Send request
	if _, err := c.conn.Write(request); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Set read timeout
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}

	// Read response header
	header := make([]byte, 7)
	if _, err := c.conn.Read(header); err != nil {
		return nil, fmt.Errorf("failed to read response header: %w", err)
	}

	// Validate response header
	respTransactionID := binary.BigEndian.Uint16(header[0:2])
	if respTransactionID != c.transactionID {
		return nil, fmt.Errorf("transaction ID mismatch: expected %d, got %d",
			c.transactionID, respTransactionID)
	}

	// Read response data
	dataLength := binary.BigEndian.Uint16(header[4:6]) - 1
	data := make([]byte, dataLength)
	if _, err := c.conn.Read(data); err != nil {
		return nil, fmt.Errorf("failed to read response data: %w", err)
	}

	// Check for exception response
	if len(data) >= 2 && data[0] >= 0x80 {
		return nil, &ModbusError{
			FunctionCode:  data[0] & 0x7F,
			ExceptionCode: data[1],
		}
	}

	return data, nil
}

// ReadCoils reads coil status (function code 0x01)
func (c *Client) ReadCoils(slaveID byte, address, quantity uint16) ([]bool, error) {
	if quantity == 0 || quantity > 2000 {
		return nil, fmt.Errorf("invalid quantity: %d (must be 1-2000)", quantity)
	}

	// Build PDU
	pdu := make([]byte, 5)
	pdu[0] = FuncCodeReadCoils
	binary.BigEndian.PutUint16(pdu[1:3], address)
	binary.BigEndian.PutUint16(pdu[3:5], quantity)

	response, err := c.sendRequest(slaveID, pdu)
	if err != nil {
		return nil, err
	}

	if len(response) < 2 {
		return nil, fmt.Errorf("invalid response length")
	}

	byteCount := response[1]
	if len(response) != int(2+byteCount) {
		return nil, fmt.Errorf("response length mismatch")
	}

	// Convert bytes to boolean array
	coils := make([]bool, quantity)
	for i := uint16(0); i < quantity; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		coils[i] = (response[2+byteIndex] & (1 << bitIndex)) != 0
	}

	return coils, nil
}

// ReadHoldingRegisters reads holding registers (function code 0x03)
func (c *Client) ReadHoldingRegisters(slaveID byte, address, quantity uint16) ([]uint16, error) {
	if quantity == 0 || quantity > 125 {
		return nil, fmt.Errorf("invalid quantity: %d (must be 1-125)", quantity)
	}

	// Build PDU
	pdu := make([]byte, 5)
	pdu[0] = FuncCodeReadHoldingRegisters
	binary.BigEndian.PutUint16(pdu[1:3], address)
	binary.BigEndian.PutUint16(pdu[3:5], quantity)

	response, err := c.sendRequest(slaveID, pdu)
	if err != nil {
		return nil, err
	}

	if len(response) < 2 {
		return nil, fmt.Errorf("invalid response length")
	}

	byteCount := response[1]
	expectedLength := quantity * 2
	if byteCount != byte(expectedLength) || len(response) != int(2+byteCount) {
		return nil, fmt.Errorf("response length mismatch")
	}

	// Convert bytes to uint16 array
	registers := make([]uint16, quantity)
	for i := uint16(0); i < quantity; i++ {
		registers[i] = binary.BigEndian.Uint16(response[2+i*2 : 4+i*2])
	}

	return registers, nil
}

// ReadInputRegisters reads input registers (function code 0x04)
func (c *Client) ReadInputRegisters(slaveID byte, address, quantity uint16) ([]uint16, error) {
	if quantity == 0 || quantity > 125 {
		return nil, fmt.Errorf("invalid quantity: %d (must be 1-125)", quantity)
	}

	// Build PDU
	pdu := make([]byte, 5)
	pdu[0] = FuncCodeReadInputRegisters
	binary.BigEndian.PutUint16(pdu[1:3], address)
	binary.BigEndian.PutUint16(pdu[3:5], quantity)

	response, err := c.sendRequest(slaveID, pdu)
	if err != nil {
		return nil, err
	}

	if len(response) < 2 {
		return nil, fmt.Errorf("invalid response length")
	}

	byteCount := response[1]
	expectedLength := quantity * 2
	if byteCount != byte(expectedLength) || len(response) != int(2+byteCount) {
		return nil, fmt.Errorf("response length mismatch")
	}

	// Convert bytes to uint16 array
	registers := make([]uint16, quantity)
	for i := uint16(0); i < quantity; i++ {
		registers[i] = binary.BigEndian.Uint16(response[2+i*2 : 4+i*2])
	}

	return registers, nil
}

// WriteSingleCoil writes a single coil (function code 0x05)
func (c *Client) WriteSingleCoil(slaveID byte, address uint16, value bool) error {
	// Build PDU
	pdu := make([]byte, 5)
	pdu[0] = FuncCodeWriteSingleCoil
	binary.BigEndian.PutUint16(pdu[1:3], address)
	if value {
		binary.BigEndian.PutUint16(pdu[3:5], 0xFF00)
	} else {
		binary.BigEndian.PutUint16(pdu[3:5], 0x0000)
	}

	response, err := c.sendRequest(slaveID, pdu)
	if err != nil {
		return err
	}

	// Verify echo response
	if len(response) != 5 || response[0] != FuncCodeWriteSingleCoil {
		return fmt.Errorf("invalid response")
	}

	return nil
}

// WriteSingleRegister writes a single register (function code 0x06)
func (c *Client) WriteSingleRegister(slaveID byte, address, value uint16) error {
	// Build PDU
	pdu := make([]byte, 5)
	pdu[0] = FuncCodeWriteSingleRegister
	binary.BigEndian.PutUint16(pdu[1:3], address)
	binary.BigEndian.PutUint16(pdu[3:5], value)

	response, err := c.sendRequest(slaveID, pdu)
	if err != nil {
		return err
	}

	// Verify echo response
	if len(response) != 5 || response[0] != FuncCodeWriteSingleRegister {
		return fmt.Errorf("invalid response")
	}

	return nil
}

// WriteMultipleCoils writes multiple coils (function code 0x0F)
func (c *Client) WriteMultipleCoils(slaveID byte, address uint16, values []bool) error {
	quantity := uint16(len(values))
	if quantity == 0 || quantity > 1968 {
		return fmt.Errorf("invalid quantity: %d (must be 1-1968)", quantity)
	}

	// Calculate byte count
	byteCount := (quantity + 7) / 8

	// Build PDU
	pdu := make([]byte, 6+byteCount)
	pdu[0] = FuncCodeWriteMultipleCoils
	binary.BigEndian.PutUint16(pdu[1:3], address)
	binary.BigEndian.PutUint16(pdu[3:5], quantity)
	pdu[5] = byte(byteCount)

	// Convert boolean array to bytes
	for i, value := range values {
		if value {
			byteIndex := i / 8
			bitIndex := i % 8
			pdu[6+byteIndex] |= 1 << bitIndex
		}
	}

	response, err := c.sendRequest(slaveID, pdu)
	if err != nil {
		return err
	}

	// Verify response
	if len(response) != 5 || response[0] != FuncCodeWriteMultipleCoils {
		return fmt.Errorf("invalid response")
	}

	return nil
}

// WriteMultipleRegisters writes multiple registers (function code 0x10)
func (c *Client) WriteMultipleRegisters(slaveID byte, address uint16, values []uint16) error {
	quantity := uint16(len(values))
	if quantity == 0 || quantity > 123 {
		return fmt.Errorf("invalid quantity: %d (must be 1-123)", quantity)
	}

	byteCount := quantity * 2

	// Build PDU
	pdu := make([]byte, 6+byteCount)
	pdu[0] = FuncCodeWriteMultipleRegisters
	binary.BigEndian.PutUint16(pdu[1:3], address)
	binary.BigEndian.PutUint16(pdu[3:5], quantity)
	pdu[5] = byte(byteCount)

	// Convert uint16 array to bytes
	for i, value := range values {
		binary.BigEndian.PutUint16(pdu[6+i*2:8+i*2], value)
	}

	response, err := c.sendRequest(slaveID, pdu)
	if err != nil {
		return err
	}

	// Verify response
	if len(response) != 5 || response[0] != FuncCodeWriteMultipleRegisters {
		return fmt.Errorf("invalid response")
	}

	return nil
}

// BatchOperation represents a batch operation
type BatchOperation struct {
	Operation string      // "read_coils", "read_holding", "read_input", "write_coils", "write_registers"
	SlaveID   byte        // Slave ID
	Address   uint16      // Starting address
	Values    interface{} // Values for write operations
	Quantity  uint16      // Quantity for read operations
}

// BatchResult represents the result of a batch operation
type BatchResult struct {
	Operation string      // Operation type
	Values    interface{} // Result values ([]bool for coils, []uint16 for registers)
	Error     error       // Error if operation failed
}

// ExecuteBatch executes multiple operations in sequence
// This provides better performance than individual calls by reusing the connection
func (c *Client) ExecuteBatch(operations []BatchOperation) []BatchResult {
	results := make([]BatchResult, len(operations))

	for i, op := range operations {
		result := BatchResult{Operation: op.Operation}

		switch op.Operation {
		case "read_coils":
			values, err := c.ReadCoils(op.SlaveID, op.Address, op.Quantity)
			result.Values = values
			result.Error = err

		case "read_holding":
			values, err := c.ReadHoldingRegisters(op.SlaveID, op.Address, op.Quantity)
			result.Values = values
			result.Error = err

		case "read_input":
			values, err := c.ReadInputRegisters(op.SlaveID, op.Address, op.Quantity)
			result.Values = values
			result.Error = err

		case "write_coils":
			if coils, ok := op.Values.([]bool); ok {
				result.Error = c.WriteMultipleCoils(op.SlaveID, op.Address, coils)
			} else {
				result.Error = fmt.Errorf("invalid values type for write_coils")
			}

		case "write_registers":
			if registers, ok := op.Values.([]uint16); ok {
				result.Error = c.WriteMultipleRegisters(op.SlaveID, op.Address, registers)
			} else {
				result.Error = fmt.Errorf("invalid values type for write_registers")
			}

		default:
			result.Error = fmt.Errorf("unknown operation: %s", op.Operation)
		}

		results[i] = result
	}

	return results
}

// ConnectionPool manages multiple Modbus connections for high-performance scenarios
type ConnectionPool struct {
	address string
	timeout time.Duration
	pool    chan *Client
	maxConn int
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(address string, maxConnections int, timeout time.Duration) (*ConnectionPool, error) {
	if maxConnections <= 0 {
		maxConnections = 10
	}
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	pool := &ConnectionPool{
		address: address,
		timeout: timeout,
		pool:    make(chan *Client, maxConnections),
		maxConn: maxConnections,
	}

	// Pre-create connections
	for i := 0; i < maxConnections; i++ {
		client, err := NewClient(ClientConfig{
			Address: address,
			Timeout: timeout,
		})
		if err != nil {
			// Close any existing connections
			pool.Close()
			return nil, fmt.Errorf("failed to create connection %d: %w", i, err)
		}
		pool.pool <- client
	}

	return pool, nil
}

// Get retrieves a connection from the pool
func (p *ConnectionPool) Get() (*Client, error) {
	select {
	case client := <-p.pool:
		return client, nil
	case <-time.After(p.timeout):
		return nil, fmt.Errorf("timeout waiting for connection")
	}
}

// Put returns a connection to the pool
func (p *ConnectionPool) Put(client *Client) {
	select {
	case p.pool <- client:
	default:
		// Pool is full, close the connection
		client.Close()
	}
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() {
	close(p.pool)
	for client := range p.pool {
		client.Close()
	}
}

// Example usage and helper functions

// ReadFloat32 reads a 32-bit float from two consecutive registers
func (c *Client) ReadFloat32(slaveID byte, address uint16, byteOrder string) (float32, error) {
	registers, err := c.ReadHoldingRegisters(slaveID, address, 2)
	if err != nil {
		return 0, err
	}

	var bits uint32
	switch byteOrder {
	case "big":
		bits = uint32(registers[0])<<16 | uint32(registers[1])
	case "little":
		bits = uint32(registers[1])<<16 | uint32(registers[0])
	default:
		return 0, fmt.Errorf("invalid byte order: %s", byteOrder)
	}

	return *(*float32)(unsafe.Pointer(&bits)), nil
}

// WriteFloat32 writes a 32-bit float to two consecutive registers
func (c *Client) WriteFloat32(slaveID byte, address uint16, value float32, byteOrder string) error {
	bits := *(*uint32)(unsafe.Pointer(&value))

	var registers []uint16
	switch byteOrder {
	case "big":
		registers = []uint16{uint16(bits >> 16), uint16(bits & 0xFFFF)}
	case "little":
		registers = []uint16{uint16(bits & 0xFFFF), uint16(bits >> 16)}
	default:
		return fmt.Errorf("invalid byte order: %s", byteOrder)
	}

	return c.WriteMultipleRegisters(slaveID, address, registers)
}
