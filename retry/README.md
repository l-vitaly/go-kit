# package retry

`package retry` provides retries to the endpoint in case of an error return.

## Usage

Max retry.

```go
func NewGRPCClient(conn *grpc.ClientConn) Service {
    var incrEndpoint endpoint.Endpoint
	{
		incrEndpoint = grpctransport.NewClient(
			conn,
			"service.Service",
			"Incr",
			encodeGRPCIncrRequest,
			decodeGRPCIncrResponse,
			pb.IncrResponse{},
			options...,
		).Endpoint()
	}
	incrEndpoint = retry.Endpoint(100*time.Millisecond)(incrEndpoint)
}
```

Callback for always retry.

```go
func NewGRPCClient(conn *grpc.ClientConn) Service {
    var incrEndpoint endpoint.Endpoint
	{
		incrEndpoint = grpctransport.NewClient(
			conn,
			"service.Service",
			"Incr",
			encodeGRPCIncrRequest,
			decodeGRPCIncrResponse,
			pb.IncrResponse{},
			options...,
		).Endpoint()
	}
	incrEndpoint = retry.WithCallbackEndpoint(100*time.Millisecond, func(n int, received error) (keepTrying bool, replacement error) { return true, nil})(incrEndpoint)
}
```
