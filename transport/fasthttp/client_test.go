package fasthttp_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	httptransport "github.com/l-vitaly/go-kit/transport/fasthttp"
	"github.com/valyala/fasthttp"
)

type TestResponse struct {
	Body []byte
}

func TestHTTPClient(t *testing.T) {
	var (
		testbody = "testbody"
		encode   = func(context.Context, *fasthttp.Request, interface{}) error { return nil }
		decode   = func(_ context.Context, r *fasthttp.Response) (interface{}, error) {
			return TestResponse{r.Body()}, nil
		}
		headers        = make(chan string, 1)
		headerKey      = "X-Foo"
		headerVal      = "abcde"
		afterHeaderKey = "X-The-Dude"
		afterHeaderVal = "Abides"
		afterVal       = ""
		afterFunc      = func(ctx context.Context, r *fasthttp.Response) context.Context {
			afterVal = string(r.Header.Peek(afterHeaderKey))
			return ctx
		}
	)

	go func() {
		fasthttp.ListenAndServe(":9000", func(rctx *fasthttp.RequestCtx) {
			headers <- string(rctx.Request.Header.Peek(headerKey))
			rctx.Response.Header.Set(afterHeaderKey, afterHeaderVal)
			rctx.SetStatusCode(http.StatusOK)
			rctx.Write([]byte(testbody))
		})
	}()

	client := httptransport.NewClient(
		"GET",
		mustParse("http://localhost:9000"),
		encode,
		decode,
		httptransport.ClientBefore(httptransport.SetRequestHeader(headerKey, headerVal)),
		httptransport.ClientAfter(afterFunc),
	)

	res, err := client.Endpoint()(context.Background(), struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	var have string
	select {
	case have = <-headers:
	case <-time.After(time.Millisecond):
		t.Fatalf("timeout waiting for %s", headerKey)
	}
	// Check that Request Header was successfully received
	if want := headerVal; want != have {
		t.Errorf("want %q, have %q", want, have)
	}

	// Check that Response header set from server was received in SetClientAfter
	if want, have := afterVal, afterHeaderVal; want != have {
		t.Errorf("want %q, have %q", want, have)
	}

	// Check that the response was successfully decoded
	response, ok := res.(TestResponse)
	if !ok {
		t.Fatal("response should be TestResponse")
	}
	if want, have := testbody, string(response.Body); want != have {
		t.Errorf("want %q, have %q", want, have)
	}
}

func TestEncodeJSONRequest(t *testing.T) {
	var header *fasthttp.RequestHeader
	var body string

	go func() {
		fasthttp.ListenAndServe(":9001", func(rctx *fasthttp.RequestCtx) {
			header = &rctx.Request.Header
			body = string(rctx.Request.Body())
		})
	}()

	client := httptransport.NewClient(
		"POST",
		mustParse("http://localhost:9001"),
		httptransport.EncodeJSONRequest,
		func(context.Context, *fasthttp.Response) (interface{}, error) { return nil, nil },
	).Endpoint()

	for _, test := range []struct {
		value interface{}
		body  string
	}{
		{nil, "null"},
		{12, "12"},
		{1.2, "1.2"},
		{true, "true"},
		{"test", "\"test\""},
		{enhancedRequest{Foo: "foo"}, "{\"foo\":\"foo\"}"},
	} {
		if _, err := client(context.Background(), test.value); err != nil {
			t.Error(err)
			continue
		}
		if body != test.body {
			t.Errorf("%v: actual %#v, expected %#v", test.value, body, test.body)
		}
	}

	if _, err := client(context.Background(), enhancedRequest{Foo: "foo"}); err != nil {
		t.Fatal(err)
	}

	if v := header.Peek("X-Edward"); v != nil {
		t.Fatalf("X-Edward value: actual %v, expected %v", nil, "Snowden")
	}
}

func mustParse(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

type enhancedRequest struct {
	Foo string `json:"foo"`
}

func (e enhancedRequest) Headers() map[string]string {
	return map[string]string{
		"X-Edward": "Snowden",
	}
}
