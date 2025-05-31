# Go Modbus Library

A high-performance, easy-to-use Modbus TCP client library for Go, supporting coil and register operations with batch processing and connection pooling.

## Features

- **Complete Modbus TCP Support**: Read/Write coils, holding registers, and input registers
- **Batch Operations**: Execute multiple operations efficiently in sequence
- **Connection Pooling**: High-performance connection management for concurrent applications
- **Type Safety**: Strong typing with proper error handling
- **Float Support**: Built-in support for 32-bit float values
- **Thread-Safe**: Safe for concurrent use with proper synchronization
- **Comprehensive Testing**: Extensive unit tests and benchmarks

## Installation

```bash
go get github.com/eddielth/modbus
```

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "log"
    "time"
    "github.com/eddielth/modbus"
)

func main() {
    // Create client
    config := modbus.ClientConfig{
        Address: "192.168.1.100:502",
        Timeout: 5 * time.Second,
    }
    
    client, err := modbus.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    slaveID := byte(1)
    
    // Read holding registers
    registers, err := client.ReadHoldingRegisters(slaveID, 0, 10)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Registers: %v\n", registers)
    
    // Write single register
    err = client.WriteSingleRegister(slaveID, 0, 1234)
    if err != nil {
        log.Fatal(err)
    }
}
```

## API Reference

### Client Operations

#### Read Operations

```go
// Read coils (discrete outputs)
coils, err := client.ReadCoils(slaveID, address, quantity)

// Read discrete inputs
inputs, err := client.ReadDiscreteInputs(slaveID, address, quantity)

// Read holding registers
registers, err := client.ReadHoldingRegisters(slaveID, address, quantity)

// Read input registers
registers, err := client.ReadInputRegisters(slaveID, address, quantity)
```

#### Write Operations

```go
// Write single coil
err := client.WriteSingleCoil(slaveID, address, true)

// Write single register
err := client.WriteSingleRegister(slaveID, address, value)

// Write multiple coils
coils := []bool{true, false, true, false}
err := client.WriteMultipleCoils(slaveID, address, coils)

// Write multiple registers
registers := []uint16{100, 200, 300, 400}
err := client.WriteMultipleRegisters(slaveID, address, registers)
```

#### Float Operations

```go
// Read 32-bit float (2 registers)
value, err := client.ReadFloat32(slaveID, address, "big")

// Write 32-bit float
err := client.WriteFloat32(slaveID, address, 3.14159, "big")
```

### Batch Operations

For better performance when executing multiple operations:

```go
operations := []modbus.BatchOperation{
    {
        Operation: "read_holding",
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

results := client.ExecuteBatch(operations)
for _, result := range results {
    if result.Error != nil {
        log.Printf("Error: %v", result.Error)
    } else {
        fmt.Printf("Success: %v\n", result.Values)
    }
}
```

### Connection Pooling

For high-performance applications with concurrent access:

```go
// Create connection pool
pool, err := modbus.NewConnectionPool("192.168.1.100:502", 10, 5*time.Second)
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

// Use in goroutines
go func() {
    client, err := pool.Get()
    if err != nil {
        return
    }
    defer pool.Put(client)
    
    // Perform operations
    registers, err := client.ReadHoldingRegisters(1, 0, 5)
    // ... handle results
}()
```

## Performance Considerations

### Connection Reuse
- Use a single client for multiple operations when possible
- Connection establishment is expensive, reuse connections
- Consider connection pooling for concurrent applications

### Batch Operations
- Use `ExecuteBatch()` for multiple operations to reduce network overhead
- Batch operations are executed sequentially but reuse the same connection

### Timeouts
- Set appropriate timeouts based on your network conditions
- Default timeout is 5 seconds
- Consider network latency and device response times

## Error Handling

The library provides detailed error information:

```go
registers, err := client.ReadHoldingRegisters(1, 0, 5)
if err != nil {
    switch e := err.(type) {
    case *modbus.ModbusError:
        // Modbus protocol exception
        fmt.Printf("Modbus exception: Function=0x%02X, Exception=0x%02X\n", 
            e.FunctionCode, e.ExceptionCode)
    default:
        // Network or other error
        fmt.Printf("Other error: %v\n", err)
    }
}
```

### Common Exception Codes
- `0x01`: Illegal Function
- `0x02`: Illegal Data Address  
- `0x03`: Illegal Data Value
- `0x04`: Slave Device Failure

## Data Type Limits

### Quantity Limits
- **Read Coils**: 1-2000 coils
- **Read Registers**: 1-125 registers
- **Write Multiple Coils**: 1-1968 coils
- **Write Multiple Registers**: 1-123 registers

### Address Range
- Addresses are 16-bit (0-65535)
- Check your device documentation for supported address ranges

## Thread Safety

- Individual client instances are thread-safe
- Multiple goroutines can safely use the same client
- Connection pools are designed for concurrent access

## Testing

Run the test suite:

```bash
go test ./...
```

Run benchmarks:

```bash
go test -bench=. ./...
```

## Examples

See the examples in the `examples/` directory for more detailed usage patterns:

- Basic operations
- Batch processing
- Connection pooling
- Error handling
- Float data types
- Real device integration

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## Performance Benchmarks

On a typical system:
- Single register read: ~1ms
- Batch of 10 operations: ~5ms
- Connection pool overhead: ~0.1ms per operation

## Compatibility

- Go 1.19+
- Modbus TCP (Modbus over TCP/IP)
- Compatible with most Modbus TCP devices and simulators

## License

MIT License - see LICENSE file for details.

## Support

For issues and questions:
- Create an issue on GitHub
- Check the examples directory
- Review the comprehensive test suite for usage patterns