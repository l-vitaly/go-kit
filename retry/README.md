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
	incrEndpoint = retry.MakeEndpoint(100*time.Millisecond, incrEndpoint, retry.Max(10))
}
```

Always retry.

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
	incrEndpoint = retry.MakeEndpoint(100*time.Millisecond, incrEndpoint, retry.Always())
}
```
