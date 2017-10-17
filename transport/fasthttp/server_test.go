package fasthttp_test

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"testing"

	httptransport "github.com/l-vitaly/go-kit/transport/fasthttp"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func TestServerBadDecode(t *testing.T) {
	s := httptransport.NewServer(
		func(context.Context, interface{}) (interface{}, error) { return struct{}{}, nil },
		func(context.Context, *fasthttp.Request) (interface{}, error) {
			return struct{}{}, errors.New("dang")
		},
		func(context.Context, *fasthttp.Response, interface{}) error { return nil },
	)

	l := fasthttputil.NewInmemoryListener()
	defer l.Close()

	go fasthttp.Serve(l, s.Handle)

	c, err := l.Dial()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if _, err = c.Write([]byte("GET / HTTP/1.1\r\nHost: aa\r\n\r\n")); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	br := bufio.NewReader(c)
	var resp fasthttp.Response
	if err = resp.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if want, have := fasthttp.StatusInternalServerError, resp.StatusCode(); want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestServerBadEndpoint(t *testing.T) {
	s := httptransport.NewServer(
		func(context.Context, interface{}) (interface{}, error) { return struct{}{}, errors.New("dang") },
		func(context.Context, *fasthttp.Request) (interface{}, error) { return struct{}{}, nil },
		func(context.Context, *fasthttp.Response, interface{}) error { return nil },
	)
	l := fasthttputil.NewInmemoryListener()
	defer l.Close()

	go fasthttp.Serve(l, s.Handle)

	c, err := l.Dial()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if _, err = c.Write([]byte("GET / HTTP/1.1\r\nHost: aa\r\n\r\n")); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	br := bufio.NewReader(c)
	var resp fasthttp.Response
	if err = resp.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if want, have := http.StatusInternalServerError, resp.StatusCode(); want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestServerBadEncode(t *testing.T) {
	s := httptransport.NewServer(
		func(context.Context, interface{}) (interface{}, error) { return struct{}{}, nil },
		func(context.Context, *fasthttp.Request) (interface{}, error) { return struct{}{}, nil },
		func(context.Context, *fasthttp.Response, interface{}) error { return errors.New("dang") },
	)

	l := fasthttputil.NewInmemoryListener()
	defer l.Close()

	go fasthttp.Serve(l, s.Handle)

	c, err := l.Dial()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if _, err = c.Write([]byte("GET / HTTP/1.1\r\nHost: aa\r\n\r\n")); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	br := bufio.NewReader(c)
	var resp fasthttp.Response
	if err = resp.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if want, have := http.StatusInternalServerError, resp.StatusCode(); want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestServerErrorEncoder(t *testing.T) {
	errTeapot := errors.New("teapot")
	code := func(err error) int {
		if err == errTeapot {
			return fasthttp.StatusTeapot
		}
		return fasthttp.StatusInternalServerError
	}
	s := httptransport.NewServer(
		func(context.Context, interface{}) (interface{}, error) { return struct{}{}, errTeapot },
		func(context.Context, *fasthttp.Request) (interface{}, error) { return struct{}{}, nil },
		func(context.Context, *fasthttp.Response, interface{}) error { return nil },
		httptransport.ServerErrorEncoder(func(_ context.Context, err error, rctx *fasthttp.Response) { rctx.SetStatusCode(code(err)) }),
	)

	l := fasthttputil.NewInmemoryListener()
	defer l.Close()

	go fasthttp.Serve(l, s.Handle)

	c, err := l.Dial()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if _, err = c.Write([]byte("GET / HTTP/1.1\r\nHost: aa\r\n\r\n")); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	br := bufio.NewReader(c)
	var resp fasthttp.Response
	if err = resp.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if want, have := http.StatusTeapot, resp.StatusCode(); want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestServerHappyPath(t *testing.T) {
	step, response := testServer(t)
	step()
	resp := <-response
	if want, have := http.StatusOK, resp.StatusCode(); want != have {
		t.Errorf("want %d, have %d (%s)", want, have, resp.Body())
	}
}

func testServer(t *testing.T) (step func(), resp <-chan *fasthttp.Response) {
	var (
		stepch   = make(chan bool)
		endpoint = func(context.Context, interface{}) (interface{}, error) { <-stepch; return struct{}{}, nil }
		response = make(chan *fasthttp.Response)
		s        = httptransport.NewServer(
			endpoint,
			func(context.Context, *fasthttp.Request) (interface{}, error) { return struct{}{}, nil },
			func(context.Context, *fasthttp.Response, interface{}) error { return nil },
			httptransport.ServerBefore(func(ctx context.Context, rctx *fasthttp.RequestCtx) context.Context { return ctx }),
			httptransport.ServerAfter(func(ctx context.Context, r *fasthttp.Response) context.Context { return ctx }),
		)
	)
	go func() {
		l := fasthttputil.NewInmemoryListener()
		defer l.Close()

		go fasthttp.Serve(l, s.Handle)

		c, err := l.Dial()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if _, err = c.Write([]byte("GET / HTTP/1.1\r\nHost: aa\r\n\r\n")); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		br := bufio.NewReader(c)
		var resp fasthttp.Response
		if err = resp.Read(br); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		response <- &resp
	}()
	return func() { stepch <- true }, response
}
