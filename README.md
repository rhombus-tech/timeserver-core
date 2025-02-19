# Timeserver Core

A distributed timestamp service providing cryptographically secure, consensus-validated timestamps for blockchain MEV prevention.

## Overview

The Timeserver Core provides a distributed network of timestamp validators that work together to create tamper-proof, globally ordered timestamps. When integrated with a blockchain, it enables mathematical prevention of MEV (Maximal Extractable Value) attacks by enforcing strict chronological ordering of transactions.

## Key Features

### MEV Prevention
- **Strict Timestamp Ordering**: Enforces chronological transaction ordering, making frontrunning mathematically impossible
- **Threshold Signatures**: Requires multiple independent validators to sign timestamps
- **Drift Protection**: Prevents timestamp manipulation through strict drift limits

### Regional Architecture
- **Geographic Distribution**: Servers organized into regions for low latency
- **Cross-Region Consistency**: Global ordering maintained while allowing regional independence
- **Latency Handling**: Configurable MaxDrift parameters account for network delays

### Fast Path Implementation
- **Quick Response**: Optimized path for rapid timestamp acquisition
- **Median Selection**: Uses median timestamp from validators to prevent outliers
- **Drift Checks**: Validates timestamps against drift limits

## Architecture

### Components
- **Consensus Module**: Implements PBFT consensus for timestamp validation
- **Region Manager**: Handles geographic distribution and server assignment
- **Fast Path**: Provides optimized timestamp acquisition
- **Network Layer**: Manages P2P communication between validators

### Security Features
- **Threshold Signatures**: Requires f+1 valid signatures in a 3f+1 system
- **Drift Protection**: Rejects timestamps outside acceptable drift windows
- **Byzantine Fault Tolerance**: Maintains correctness with up to f Byzantine validators

## Integration

### Blockchain Integration
1. Request timestamp from timeserver network
2. Include signed timestamp in transaction
3. Blockchain enforces strict ordering based on timestamps

### MEV Prevention
- Transactions must include valid timestamps
- Block producers must respect timestamp ordering
- Higher gas fees cannot override chronological order

## Configuration

### Regional Setup
```yaml
regions:
  - id: "us-east"
    name: "US East"
    servers: ["validator1", "validator2", "validator3"]
  - id: "eu-west"
    name: "Europe West"
    servers: ["validator4", "validator5", "validator6"]
```

### Drift Parameters
```yaml
consensus:
  maxDrift: "100ms"      # Maximum drift between validators
  futureWindow: "500ms"  # Maximum future timestamp allowed
  minValidators: 4       # Minimum validators for consensus
```

## Getting Started

1. Configure validators and regions
2. Start timeserver network
3. Integrate with blockchain
4. Enforce timestamp ordering

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and development process.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.