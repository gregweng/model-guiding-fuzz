# Fuzzing Framework Customization Guide

This document provides guidance on how to adapt the fuzzing framework for testing different services. The framework can be customized for both gRPC services and PubSub-based services.

## File Structure Overview

Key files that require modification:

```
├── types.go           # Core types and interfaces
├── fuzzer.go         # Main fuzzing engine
├── guider.go         # Coverage tracking
├── main.go           # Entry point and configuration
└── your_service/     # Your service-specific implementation
```

## Customization for gRPC Services

### Message Types
- **File**: `your_service/proto/service.proto`
- **What to Create**: New protobuf message definitions
- **Example**:
```proto
message YourServiceRequest {
    string field1 = 1;
    int32 field2 = 2;
}
```

### Environment Setup
- **File**: `your_service/environment.go`
- **Components to Implement**: 
  - Service environment struct
  - Setup/teardown methods
  - State management
- **Example**:
```go
type YourServiceEnvironment struct {
    clients []*grpc.ClientConn
    servers []*grpc.Server
    config ServiceConfig
}
```

### Event Types
- **File**: `types.go`
- **Components to Modify**: 
  - `Event` struct (lines 8-13)
  - `SchedulingChoiceType` constants (lines 16-21)
- **Example**:
```go
type Event struct {
    Name string  // e.g., "RequestSent", "ResponseReceived"
    ServiceID string
    Params map[string]interface{}
}
```

### Scheduling Choices
- **File**: `types.go`
- **Components to Modify**: 
  - `SchedulingChoice` struct (lines 26-36)
  - Add new choice types
- **Example**:
```go
type SchedulingChoice struct {
    Type SchedulingChoiceType
    ServiceID string
    Operation string
    Params map[string]interface{}
}
```

### Fuzzer Config
- **File**: `fuzzer.go`
- **Components to Modify**: 
  - `FuzzerConfig` struct
  - `NewFuzzer` function
- **Example**:
```go
type FuzzerConfig struct {
    NumServices int
    RequestTimeout time.Duration
    MaxConcurrentRequests int
}
```

### State Checker
- **File**: `checker.go`
- **Components to Implement**: 
  - New checker function
  - State verification logic
- **Example**:
```go
func NewServiceChecker() func(*ServiceEnvironment) bool {
    return func(env *ServiceEnvironment) bool {
        // Verify service invariants
        return true
    }
}
```

### Guider
- **File**: `guider.go`
- **Components to Modify**: 
  - Implement new `Guider` interface (lines 23-27)
  - Create service-specific guider
- **Example**:
```go
func (g *ServiceGuider) Check(trace *List[*Event]) {
    // Track API coverage
    // Track error scenarios
}
```

### Mutator
- **File**: `fuzzer.go`
- **Components to Modify**: 
  - Implement `Mutator` interface
  - Add service-specific mutation strategies
- **Example**:
```go
type ServiceMutator struct {
    // Strategies for mutating API calls
    // Timing variations
    // Error injections
}
```

## Customization for PubSub Services

### Message Types
- **File**: `your_service/pubsub/types.go`
- **Components to Create**: 
  - Message type definitions
  - Topic structures
- **Example**:
```go
type PubSubMessage struct {
    Topic string
    Payload []byte
    Attributes map[string]string
}
```

### Environment Setup
- **File**: `your_service/pubsub/environment.go`
- **Components to Implement**: 
  - PubSub environment
  - Publisher/Subscriber management
- **Example**:
```go
type PubSubEnvironment struct {
    publishers map[string]*pubsub.Publisher
    subscribers map[string]*pubsub.Subscriber
    topics []string
}
```

### Event Types
- **File**: `types.go`
- **Components to Add**: 
  - PubSub-specific event types
  - Event tracking structures
- **Example**:
```go
const (
    EventPublish = "Publish"
    EventSubscribe = "Subscribe"
    EventAck = "Acknowledge"
    EventNack = "NotAcknowledge"
)
```

### Message Queue
- **File**: `types.go`
- **Components to Modify**: 
  - `Queue` implementation (lines 52-87)
  - Add topic-based routing
- **Example**:
```go
type TopicQueue struct {
    messages map[string][]*PubSubMessage  // by topic
    subscribers map[string][]string  // topic -> subscriber IDs
}
```

### Fuzzer Config
- **File**: `fuzzer.go`
- **Components to Add**: 
  - PubSub-specific configuration
  - Topic management settings
- **Example**:
```go
type PubSubFuzzerConfig struct {
    NumTopics int
    NumSubscribers int
    MessageRetention time.Duration
    MaxOutstanding int
}
```

### State Checker
- **File**: `checker.go`
- **Components to Implement**: 
  - Message delivery verification
  - Ordering guarantees checking
- **Example**:
```go
func NewPubSubChecker() func(*PubSubEnvironment) bool {
    return func(env *PubSubEnvironment) bool {
        // Verify message delivery
        // Check ordering guarantees
        return true
    }
}
```

### Scheduling Choices
- **File**: `types.go`
- **Components to Add**: 
  - PubSub scheduling types
  - Message delivery choices
- **Example**:
```go
type PubSubSchedulingChoice struct {
    Type string  // "Publish", "Deliver", "Ack"
    Topic string
    SubscriberID string
    MessageID string
}
```

### Guider
- **File**: `guider.go`
- **Components to Implement**: 
  - PubSub-specific coverage tracking
  - Message flow analysis
- **Example**:
```go
type PubSubGuider struct {
    deliveryPatterns map[string]bool
    orderingScenarios map[string]bool
    errorCases map[string]bool
}
```

[Rest of the document remains the same...]