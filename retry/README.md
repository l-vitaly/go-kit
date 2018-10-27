# package retry

`package retry` provides retries to the endpoint in case of an error return.

## Usage

A simple max retry.

```go
...

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
	incrEndpoint = retry.Retry(100*time.Millisecond, incrEndpoint, MaxRetries(10))
}
...
```
