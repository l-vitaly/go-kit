package wsjsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-kit/kit/log/level"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/websocket"
)

type requestIDKeyType struct{}

var RequestIDKey requestIDKeyType

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512

	workersDefault = 100

	workerBufferDefault = 20
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type wsClient struct {
	ctx       context.Context
	s         *Server
	conn      *websocket.Conn
	send      chan []byte
	stream    map[string]*Stream
	streamMux sync.RWMutex
}

func (c *wsClient) readPump() {
	defer func() {
		c.s.unregister <- c
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				_ = level.Error(c.s.logger).Log("err", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		result, isBatch, err := c.s.rpcCall(c.ctx, c, message, false)
		if err != nil {
			_ = c.s.logger.Log("err", err)
			c.send <- c.s.marshalResponse([]Response{c.s.errorEncoder(c.ctx, err)}, isBatch)
			continue
		}

		c.send <- c.s.marshalResponse(result, isBatch)
	}
}

func (c *wsClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write(newline)
				_, _ = w.Write(<-c.send)
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type Stream struct {
	streamRead chan []byte
	reqID      *RequestID
	c          *wsClient
}

func (s *Stream) Read() chan []byte {
	return s.streamRead
}

func (s *Stream) Write(v interface{}) error {
	result, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.c.send <- s.c.s.marshalResponse([]Response{{
		JSONRPC: "2.0",
		Result:  result,
		ID:      s.reqID,
		Stream:  true,
	}}, false)
	return nil
}

// Server wraps an endpoint and implements http.Handler.
type Server struct {
	upgrader     websocket.Upgrader
	ecm          EndpointCodecMap
	ecms         EndpointCodecStreamMap
	before       []httptransport.RequestFunc
	errorEncoder ErrorEncoder
	workers      int
	workerBuffer int

	clients    map[*wsClient]bool
	register   chan *wsClient
	unregister chan *wsClient

	logger log.Logger
}

// NewServer constructs a new server, which implements http.Server.
func NewServer(
	ecm EndpointCodecMap,
	ecms EndpointCodecStreamMap,
	options ...ServerOption,
) *Server {
	s := &Server{
		ecm:          ecm,
		ecms:         ecms,
		errorEncoder: DefaultErrorEncoder,
		logger:       log.NewNopLogger(),
		workers:      workersDefault,
		workerBuffer: workerBufferDefault,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		register:   make(chan *wsClient),
		unregister: make(chan *wsClient),
		clients:    make(map[*wsClient]bool),
	}
	for _, option := range options {
		option(s)
	}
	go s.run()
	return s
}

// ServerOption sets an optional parameter for servers.
type ServerOption func(*Server)

// ServerBefore functions are executed on the HTTP request object before the
// request is decoded.
func ServerBefore(before ...httptransport.RequestFunc) ServerOption {
	return func(s *Server) { s.before = append(s.before, before...) }
}

// ServerAfter functions are executed on the HTTP response writer after the
// endpoint is invoked, but before anything is written to the client.
//func ServerAfter(after ...httptransport.ServerResponseFunc) ServerOption {
//	return func(s *Server) { s.after = append(s.after, after...) }
//}

// ServerErrorEncoder is used to encode errors to the http.ResponseWriter
// whenever they're encountered in the processing of a request. Clients can
// use this to provide custom error formatting and response codes. By default,
// errors will be written with the DefaultErrorEncoder.
func ServerErrorEncoder(ee ErrorEncoder) ServerOption {
	return func(s *Server) { s.errorEncoder = ee }
}

func Workers(workers int) ServerOption {
	return func(s *Server) { s.workers = workers }
}

// ServerErrorLogger is used to log non-terminal errors. By default, no errors
// are logged. This is intended as a diagnostic measure. Finer-grained control
// of error handling, including logging in more detail, should be performed in a
// custom ServerErrorEncoder or ServerFinalizer, both of which have access to
// the context.
func ServerErrorLogger(logger log.Logger) ServerOption {
	return func(s *Server) { s.logger = logger }
}

// ErrorEncoder is responsible for encoding an error to the ResponseWriter.
// Users are encouraged to use custom ErrorEncoders to encode HTTP errors to
// their clients, and will likely want to pass and check for their own error
// types. See the example shipping/handling service.
type ErrorEncoder func(ctx context.Context, err error) Response

// ResponseWriter ...
type ResponseWriter func(ctx context.Context, responses []Response, isBatch bool, w http.ResponseWriter)

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	for _, f := range s.before {
		ctx = f(ctx, r)
	}
	// Upgrade the incoming HTTP request to a WebSocket connection
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		_ = s.logger.Log("err", err)
		return
	}

	c := &wsClient{ctx: ctx, s: s, conn: conn, send: make(chan []byte, 256), stream: map[string]*Stream{}}

	s.register <- c

	go c.writePump()
	go c.readPump()
}

func (s *Server) marshalResponse(responses []Response, isBatch bool) (data []byte) {
	if !isBatch && len(responses) > 0 {
		data, _ = json.Marshal(responses[0])
		return
	}
	data, _ = json.Marshal(responses)
	return
}

func (s *Server) requestWorker(ctx context.Context, c *wsClient, requests chan Request, responses chan Response) {
	for req := range requests {

		c.streamMux.Lock()
		if stream, ok := c.stream[req.Method]; ok {
			c.streamMux.Unlock()
			stream.streamRead <- req.Params
			continue
		}
		c.streamMux.Unlock()

		ctx = context.WithValue(ctx, RequestIDKey, req.ID)
		// Get the endpoint and codecs from the map using the method
		// defined in the JSON  object
		ecm, ok := s.ecm[req.Method]
		if !ok {
			if ecm, ok := s.ecms[req.Method]; ok {
				stream := &Stream{
					reqID:      req.ID,
					c:          c,
					streamRead: make(chan []byte),
				}

				c.streamMux.Lock()
				c.stream[req.Method] = stream
				c.streamMux.Unlock()

				// Decode the JSON "params"
				reqParams, err := ecm.Decode(ctx, req.Params, stream)
				if err != nil {
					_ = s.logger.Log("err", err)
					responses <- s.errorEncoder(ctx, err)
					continue
				}
				go func() {
					responses <- Response{
						ID:      req.ID,
						JSONRPC: Version,
						Stream:  true,
					}
					_, _ = ecm.Endpoint(ctx, reqParams)
				}()
				continue
			} else {
				err := methodNotFoundError(fmt.Sprintf("Method %s was not found.", req.Method))
				_ = s.logger.Log("err", err)
				responses <- s.errorEncoder(ctx, err)
				continue
			}
		}

		// Decode the JSON "params"
		reqParams, err := ecm.Decode(ctx, req.Params)
		if err != nil {
			_ = s.logger.Log("err", err)
			responses <- s.errorEncoder(ctx, err)
			continue
		}

		response, err := ecm.Endpoint(ctx, reqParams)
		if err != nil {
			_ = s.logger.Log("err", err)
			responses <- s.errorEncoder(ctx, err)
			return
		}
		res := Response{
			ID:      req.ID,
			JSONRPC: Version,
		}
		// Encode the response from the Endpoint
		resParams, err := ecm.Encode(ctx, response)
		if err != nil {
			_ = s.logger.Log("err", err)
			responses <- s.errorEncoder(ctx, err)
			return
		}
		res.Result = resParams
		responses <- res
	}
}

func (s *Server) rpcCall(ctx context.Context, c *wsClient, data []byte, async bool) (result []Response, isBatch bool, err error) {
	isBatch = true
	if len(data) > 0 && !bytes.HasPrefix(data, []byte("[")) && !bytes.HasSuffix(data, []byte("]")) {
		isBatch = false
		buf := new(bytes.Buffer)
		buf.WriteString("[")
		buf.Write(data)
		buf.WriteString("]")
		data = buf.Bytes()
	}

	// Decode the body into an object
	var reqs []Request
	err = json.Unmarshal(data, &reqs)
	if err != nil {
		return nil, false, err
	}

	requests := make(chan Request, s.workerBuffer)
	responses := make(chan Response, s.workerBuffer)

	if async {
		go s.requestWorker(ctx, c, requests, responses)
	} else {
		for w := 1; w <= s.workers; w++ {
			go s.requestWorker(ctx, c, requests, responses)
		}
	}

	for _, req := range reqs {
		requests <- req
	}
	close(requests)

	for i := 0; i < len(reqs); i++ {
		result = append(result, <-responses)
	}
	return
}

func (s *Server) run() {
	for {
		select {
		case client := <-s.register:
			s.clients[client] = true
		case client := <-s.unregister:
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
			}
		}
	}
}

// DefaultErrorEncoder writes the error to the ResponseWriter,
// as a json-rpc error response, with an InternalError status code.
// The Error() string of the error will be used as the response error message.
// If the error implements ErrorCoder, the provided code will be set on the
// response error.
// If the error implements Headerer, the given headers will be set.
func DefaultErrorEncoder(ctx context.Context, err error) Response {
	e := Error{
		Code:    InternalError,
		Message: err.Error(),
	}
	if sc, ok := err.(ErrorCoder); ok {
		e.Code = sc.ErrorCode()
	}

	if sc, ok := err.(ErrorData); ok {
		e.Data = sc.ErrorData()
	}

	var requestID *RequestID
	if v := ctx.Value(RequestIDKey); v != nil {
		requestID = v.(*RequestID)
	}

	return Response{
		ID:      requestID,
		JSONRPC: Version,
		Error:   &e,
	}
}

// ErrorCoder is checked by DefaultErrorEncoder. If an error value implements
// ErrorCoder, the integer result of ErrorCode() will be used as the JSONRPC
// error code when encoding the error.
//
// By default, InternalError (-32603) is used.
type ErrorCoder interface {
	ErrorCode() int
}

// ErrorData is checked by DefaultErrorEncoder. If an error value implements
// ErrorData, the interface{} result of ErrorData() will be used as the JSONRPC
// error data when encoding the error.
//
// By default, empty is used.
type ErrorData interface {
	ErrorData() interface{}
}

// interceptingWriter intercepts calls to WriteHeader, so that a finalizer
// can be given the correct status code.
type interceptingWriter struct {
	http.ResponseWriter
	code int
}

// WriteHeader may not be explicitly called, so care must be taken to
// initialize w.code to its default value of http.StatusOK.
func (w *interceptingWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func reqID2Str(id *RequestID) string {
	if s, err := id.String(); err == nil {
		return s
	}
	if i, err := id.Int(); err != nil {
		return strconv.Itoa(i)
	}
	f, _ := id.Float32()
	return strconv.FormatFloat(float64(f), 'f', 2, 32)
}
