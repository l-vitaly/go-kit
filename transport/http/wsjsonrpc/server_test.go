package wsjsonrpc_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/l-vitaly/go-kit/transport/http/wsjsonrpc"
)

func addBody() io.Reader {
	return body(`{"jsonrpc": "2.0", "method": "add", "params": [3, 2], "id": 1}`)
}

func body(in string) io.Reader {
	return strings.NewReader(in)
}

func unmarshalResponse(body []byte) (resp wsjsonrpc.Response, err error) {
	err = json.Unmarshal(body, &resp)
	return
}

func unmarshalResponses(body []byte) (resp []wsjsonrpc.Response, err error) {
	err = json.Unmarshal(body, &resp)
	return
}

func expectErrorCode(t *testing.T, want int, body []byte) {
	t.Helper()

	r, err := unmarshalResponse(body)
	if err != nil {
		t.Fatalf("Can't decode response: %v (%s)", err, body)
	}
	if r.Error == nil {
		t.Fatalf("Expected error on response. Got none: %s", body)
	}
	if have := r.Error.Code; want != have {
		t.Fatalf("Unexpected error code. Want %d, have %d: %s", want, have, body)
	}
}

func expectValidRequestID(t *testing.T, want int, body []byte) {
	t.Helper()

	r, err := unmarshalResponse(body)
	if err != nil {
		t.Fatalf("Can't decode response: %v (%s)", err, body)
	}
	have, err := r.ID.Int()
	if err != nil {
		t.Fatalf("Can't get requestID in response. err=%s, body=%s", err, body)
	}
	if want != have {
		t.Fatalf("Request ID: want %d, have %d (%s)", want, have, body)
	}
}

func expectNilRequestID(t *testing.T, body []byte) {
	t.Helper()

	r, err := unmarshalResponse(body)
	if err != nil {
		t.Fatalf("Can't decode response: %v (%s)", err, body)
	}
	if r.ID != nil {
		t.Fatalf("Request ID: want nil, have %v", r.ID)
	}
}

func nopDecoder(context.Context, json.RawMessage, *websocket.Conn) (interface{}, error) {
	return struct{}{}, nil
}
func nopEncoder(context.Context, interface{}) (json.RawMessage, error) { return []byte("[]"), nil }

type mockLogger struct {
	Called   bool
	LastArgs []interface{}
}

func (l *mockLogger) Log(keyvals ...interface{}) error {
	l.Called = true
	l.LastArgs = append(l.LastArgs, keyvals)
	return nil
}

//func TestServerBadDecode(t *testing.T) {
//	ecm := wsjsonrpc.EndpointCodecMap{
//		"add": wsjsonrpc.EndpointCodec{
//			Endpoint: endpoint.Nop,
//			Decode: func(context.Context, json.RawMessage, *websocket.Conn) (interface{}, error) {
//				return struct{}{}, errors.New("oof")
//			},
//			Encode: nopEncoder,
//		},
//	}
//	logger := mockLogger{}
//	handler := wsjsonrpc.NewServer(ecm, wsjsonrpc.ServerErrorLogger(&logger))
//	server := httptest.NewServer(handler)
//	defer server.Close()
//	resp, _ := http.Post(server.URL, "application/json", addBody())
//	buf, _ := ioutil.ReadAll(resp.Body)
//	if want, have := http.StatusOK, resp.StatusCode; want != have {
//		t.Errorf("want %d, have %d: %s", want, have, buf)
//	}
//	expectErrorCode(t, wsjsonrpc.InternalError, buf)
//	if !logger.Called {
//		t.Fatal("Expected logger to be called with error. Wasn't.")
//	}
//}
//
//func TestServerBadEndpoint(t *testing.T) {
//	ecm := wsjsonrpc.EndpointCodecMap{
//		"add": wsjsonrpc.EndpointCodec{
//			Endpoint: func(context.Context, interface{}) (interface{}, error) { return struct{}{}, errors.New("oof") },
//			Decode:   nopDecoder,
//			Encode:   nopEncoder,
//		},
//	}
//	handler := wsjsonrpc.NewServer(ecm)
//	server := httptest.NewServer(handler)
//	defer server.Close()
//	resp, _ := http.Post(server.URL, "application/json", addBody())
//	if want, have := http.StatusOK, resp.StatusCode; want != have {
//		t.Errorf("want %d, have %d", want, have)
//	}
//	buf, _ := ioutil.ReadAll(resp.Body)
//	t.Log(string(buf))
//	expectErrorCode(t, wsjsonrpc.InternalError, buf)
//	expectValidRequestID(t, 1, buf)
//}
//
//func TestServerBadEncode(t *testing.T) {
//	ecm := wsjsonrpc.EndpointCodecMap{
//		"add": wsjsonrpc.EndpointCodec{
//			Endpoint: endpoint.Nop,
//			Decode:   nopDecoder,
//			Encode:   func(context.Context, interface{}) (json.RawMessage, error) { return []byte{}, errors.New("oof") },
//		},
//	}
//	handler := wsjsonrpc.NewServer(ecm)
//	server := httptest.NewServer(handler)
//	defer server.Close()
//	resp, _ := http.Post(server.URL, "application/json", addBody())
//	if want, have := http.StatusOK, resp.StatusCode; want != have {
//		t.Errorf("want %d, have %d", want, have)
//	}
//	buf, _ := ioutil.ReadAll(resp.Body)
//	expectErrorCode(t, jsonrpc.InternalError, buf)
//	expectValidRequestID(t, 1, buf)
//}
//
//func TestCanRejectNonPostRequest(t *testing.T) {
//	ecm := jsonrpc.EndpointCodecMap{}
//	handler := jsonrpc.NewServer(ecm)
//	server := httptest.NewServer(handler)
//	defer server.Close()
//	resp, _ := http.Get(server.URL)
//	if want, have := http.StatusMethodNotAllowed, resp.StatusCode; want != have {
//		t.Errorf("want %d, have %d", want, have)
//	}
//}
//
//func TestCanRejectInvalidJSON(t *testing.T) {
//	ecm := jsonrpc.EndpointCodecMap{}
//	handler := jsonrpc.NewServer(ecm)
//	server := httptest.NewServer(handler)
//	defer server.Close()
//	resp, _ := http.Post(server.URL, "application/json", body("clearlynotjson"))
//	if want, have := http.StatusOK, resp.StatusCode; want != have {
//		t.Errorf("want %d, have %d", want, have)
//	}
//	buf, _ := ioutil.ReadAll(resp.Body)
//	expectErrorCode(t, jsonrpc.ParseError, buf)
//	expectNilRequestID(t, buf)
//}
//
//func TestServerUnregisteredMethod(t *testing.T) {
//	ecm := jsonrpc.EndpointCodecMap{}
//	handler := jsonrpc.NewServer(ecm)
//	server := httptest.NewServer(handler)
//	defer server.Close()
//	resp, _ := http.Post(server.URL, "application/json", addBody())
//	if want, have := http.StatusOK, resp.StatusCode; want != have {
//		t.Errorf("want %d, have %d", want, have)
//	}
//	buf, _ := ioutil.ReadAll(resp.Body)
//	expectErrorCode(t, jsonrpc.MethodNotFoundError, buf)
//}
//
//func TestServerHappyPath(t *testing.T) {
//	step, response := testServer(t)
//	step()
//	resp := <-response
//
//	defer resp.Body.Close() // nolint
//	buf, _ := ioutil.ReadAll(resp.Body)
//	if want, have := http.StatusOK, resp.StatusCode; want != have {
//		t.Errorf("want %d, have %d (%s)", want, have, buf)
//	}
//	r, err := unmarshalResponse(buf)
//	if err != nil {
//		t.Fatalf("Can't decode response. err=%s, body=%s", err, buf)
//	}
//	if r.JSONRPC != jsonrpc.Version {
//		t.Fatalf("JSONRPC Version: want=%s, got=%s", jsonrpc.Version, r.JSONRPC)
//	}
//	if r.Error != nil {
//		t.Fatalf("Unxpected error on response: %s", buf)
//	}
//}
//
//func TestServerBatchAsyncHappyPath(t *testing.T) {
//	step, response := testServerForBatch(t, true)
//	step()
//	step()
//	resp := <-response
//
//	defer resp.Body.Close() // nolint
//	buf, _ := ioutil.ReadAll(resp.Body)
//	if want, have := http.StatusOK, resp.StatusCode; want != have {
//		t.Errorf("want %d, have %d (%s)", want, have, buf)
//	}
//	res, err := unmarshalResponses(buf)
//	if err != nil {
//		t.Fatalf("Can't decode response. err=%s, body=%s", err, buf)
//	}
//
//	for _, r := range res {
//		if r.JSONRPC != jsonrpc.Version {
//			t.Fatalf("JSONRPC Version: want=%s, got=%s", jsonrpc.Version, r.JSONRPC)
//		}
//		if r.Error != nil {
//			t.Fatalf("Unxpected error on response: %s", buf)
//		}
//	}
//}
//
//func TestServerBatchHappyPath(t *testing.T) {
//	step, response := testServerForBatch(t, false)
//	step()
//	step()
//	resp := <-response
//
//	defer resp.Body.Close() // nolint
//	buf, _ := ioutil.ReadAll(resp.Body)
//	if want, have := http.StatusOK, resp.StatusCode; want != have {
//		t.Errorf("want %d, have %d (%s)", want, have, buf)
//	}
//	res, err := unmarshalResponses(buf)
//	if err != nil {
//		t.Fatalf("Can't decode response. err=%s, body=%s", err, buf)
//	}
//	for i, r := range res {
//		id, err := r.ID.Int()
//		if err != nil {
//			t.Fatal(err)
//		}
//		if id != i+1 {
//			t.Fatalf("JSONRPC ID: want=%d, got=%d", i+1, id)
//		}
//		if r.JSONRPC != jsonrpc.Version {
//			t.Fatalf("JSONRPC Version: want=%s, got=%s", jsonrpc.Version, r.JSONRPC)
//		}
//		if r.Error != nil {
//			t.Fatalf("Unxpected error on response: %s", buf)
//		}
//	}
//}
//
//func TestMultipleServerBefore(t *testing.T) {
//	var done = make(chan struct{})
//	ecm := wsjsonrpc.EndpointCodecMap{
//		"add": wsjsonrpc.EndpointCodec{
//			Endpoint: endpoint.Nop,
//			Decode:   nopDecoder,
//			Encode:   nopEncoder,
//		},
//	}
//	handler := wsjsonrpc.NewServer(
//		ecm,
//		wsjsonrpc.ServerBefore(func(ctx context.Context, r *http.Request) context.Context {
//			ctx = context.WithValue(ctx, "one", 1)
//
//			return ctx
//		}),
//		wsjsonrpc.ServerBefore(func(ctx context.Context, r *http.Request) context.Context {
//			if _, ok := ctx.Value("one").(int); !ok {
//				t.Error("Value was not set properly when multiple ServerBefores are used")
//			}
//
//			close(done)
//			return ctx
//		}),
//	)
//	server := httptest.NewServer(handler)
//	defer server.Close()
//	http.Post(server.URL, "application/json", addBody()) // nolint
//
//	select {
//	case <-done:
//	case <-time.After(time.Second):
//		t.Fatal("timeout waiting for finalizer")
//	}
//}

//func TestMultipleServerAfter(t *testing.T) {
//	var done = make(chan struct{})
//	ecm := wsjsonrpc.EndpointCodecMap{
//		"add": wsjsonrpc.EndpointCodec{
//			Endpoint: endpoint.Nop,
//			Decode:   nopDecoder,
//			Encode:   nopEncoder,
//		},
//	}
//	handler := wsjsonrpc.NewServer(
//		ecm,
//		wsjsonrpc.ServerAfter(func(ctx context.Context, w http.ResponseWriter) context.Context {
//			ctx = context.WithValue(ctx, "one", 1)
//
//			return ctx
//		}),
//        wsjsonrpc.ServerAfter(func(ctx context.Context, w http.ResponseWriter) context.Context {
//			if _, ok := ctx.Value("one").(int); !ok {
//				t.Error("Value was not set properly when multiple ServerAfters are used")
//			}
//
//			close(done)
//			return ctx
//		}),
//	)
//	server := httptest.NewServer(handler)
//	defer server.Close()
//	http.Post(server.URL, "application/json", addBody()) // nolint
//
//	select {
//	case <-done:
//	case <-time.After(time.Second):
//		t.Fatal("timeout waiting for finalizer")
//	}
//}

//func TestCanFinalize(t *testing.T) {
//	var done = make(chan struct{})
//	var finalizerCalled bool
//	ecm := wsjsonrpc.EndpointCodecMap{
//		"add": wsjsonrpc.EndpointCodec{
//			Endpoint: endpoint.Nop,
//			Decode:   nopDecoder,
//			Encode:   nopEncoder,
//		},
//	}
//	handler := jsonrpc.NewServer(
//		ecm,
//		jsonrpc.ServerFinalizer(func(ctx context.Context, code int, req *http.Request) {
//			finalizerCalled = true
//			close(done)
//		}),
//	)
//	server := httptest.NewServer(handler)
//	defer server.Close()
//	http.Post(server.URL, "application/json", addBody()) // nolint
//
//	select {
//	case <-done:
//	case <-time.After(time.Second):
//		t.Fatal("timeout waiting for finalizer")
//	}
//
//	if !finalizerCalled {
//		t.Fatal("Finalizer was not called.")
//	}
//}

func TestServer(t *testing.T) {
	endpoint := func(ctx context.Context, request interface{}) (response interface{}, err error) {
		stream := request.(*wsjsonrpc.Stream)

		var i int

		for {
			_ = stream.Write(map[string]interface{}{"msg": "ping"})

			time.Sleep(time.Second)
			i++

			if i >= 5 {
				break
			}
		}

		return struct{}{}, nil
	}
	ecms := wsjsonrpc.EndpointCodecStreamMap{
		"add": wsjsonrpc.EndpointCodecStream{
			Endpoint: endpoint,
			Decode: func(ctx context.Context, message json.RawMessage, stream *wsjsonrpc.Stream) (request interface{}, err error) {
				return stream, nil
			},
		},
	}

	handler := wsjsonrpc.NewServer(wsjsonrpc.EndpointCodecMap{}, ecms)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("could not open a ws connection on %s %v", wsURL, err)
	}
	defer ws.Close()

	wait := make(chan struct{})

	go func() {
		var i int
		for {
			var resp map[string]interface{}
			if err := ws.ReadJSON(&resp); err != nil {
				t.Error(err)
			}

			fmt.Println(resp)

			i++

			if i >= 6 {
				close(wait)
				return
			}
		}
	}()

	if err := ws.WriteMessage(websocket.TextMessage, []byte(`{"jsonrpc": "2.0", "id": 1, "method": "add"}`)); err != nil {
		t.Fatalf("could not send message over ws connection %v", err)
	}

	<-wait

}

//func testServer(t *testing.T) (step func(), resp <-chan *http.Response) {
//	var (
//		stepch   = make(chan bool)
//		endpoint = func(ctx context.Context, request interface{}) (response interface{}, err error) {
//			<-stepch
//			return struct{}{}, nil
//		}
//		response = make(chan *http.Response)
//		ecm      = wsjsonrpc.EndpointCodecMap{
//			"add": wsjsonrpc.EndpointCodec{
//				Endpoint: endpoint,
//				Decode:   nopDecoder,
//				Encode:   nopEncoder,
//			},
//		}
//		handler = wsjsonrpc.NewServer(ecm)
//	)
//	go func() {
//		server := httptest.NewServer(handler)
//		defer server.Close()
//		rb := strings.NewReader(`{"jsonrpc": "2.0", "method": "add", "params": [3, 2], "id": 1}`)
//		resp, err := http.Post(server.URL, "application/json", rb)
//		if err != nil {
//			t.Error(err)
//			return
//		}
//		response <- resp
//	}()
//	return func() { stepch <- true }, response
//}
//
//func testServerForBatch(t *testing.T, async bool) (step func(), resp <-chan *http.Response) {
//	var (
//		stepch      = make(chan bool)
//		endpointAdd = func(ctx context.Context, request interface{}) (response interface{}, err error) {
//			<-stepch
//			return "add", nil
//		}
//		endpointDelete = func(ctx context.Context, request interface{}) (response interface{}, err error) {
//			<-stepch
//			return "delete", nil
//		}
//		response = make(chan *http.Response)
//		ecm      = wsjsonrpc.EndpointCodecMap{
//			"add": wsjsonrpc.EndpointCodec{
//				Endpoint: endpointAdd,
//				Decode:   nopDecoder,
//				Encode:   nopEncoder,
//			},
//			"delete": wsjsonrpc.EndpointCodec{
//				Endpoint: endpointDelete,
//				Decode:   nopDecoder,
//				Encode:   nopEncoder,
//			},
//		}
//		handler = wsjsonrpc.NewServer(ecm)
//	)
//	go func() {
//		server := httptest.NewServer(handler)
//		defer server.Close()
//		rb := strings.NewReader(`[{"jsonrpc": "2.0", "method": "add", "params": [3, 2], "id": 1}, {"jsonrpc": "2.0", "method": "delete", "params": [3, 2], "id": 2}]`)
//
//		req, err := http.NewRequest(http.MethodPost, server.URL, rb)
//		if err != nil {
//			t.Error(err)
//			return
//		}
//		req.Header.Set("Content-Type", "application/json")
//		if async {
//			req.Header.Set("X-Async", "on")
//		}
//		resp, err := http.DefaultClient.Do(req)
//		if err != nil {
//			t.Error(err)
//			return
//		}
//		response <- resp
//	}()
//	return func() { stepch <- true }, response
//}
