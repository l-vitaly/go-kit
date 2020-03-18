package jsonrpc

import (
	"errors"

	"github.com/pquerna/ffjson/ffjson"
)

// RawMessage is a raw encoded JSON value.
// It implements Marshaler and Unmarshaler and can
// be used to delay JSON decoding or precomputed a JSON encoding.
// ffjson: skip
type RawMessage []byte

// MarshalJSON returns m as the JSON encoding of m.
func (m RawMessage) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return m, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (m *RawMessage) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("ffjson.RawMessage: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

// Request defines a JSON RPC request from the spec
// http://www.jsonrpc.org/specification#request_object
type Request struct {
	JSONRPC string     `json:"jsonrpc"`
	Method  string     `json:"method"`
	Params  RawMessage `json:"params"`
	ID      *RequestID `json:"id"`
}

// RequestID defines a request ID that can be string, number, or null.
// An identifier established by the Client that MUST contain a String,
// Number, or NULL value if included.
// If it is not included it is assumed to be a notification.
// The value SHOULD normally not be Null and
// Numbers SHOULD NOT contain fractional parts.
// ffjson: skip
type RequestID struct {
	intValue    int
	intError    error
	floatValue  float32
	floatError  error
	stringValue string
	stringError error
}

// UnmarshalJSON satisfies ffjson.Unmarshaler
func (id *RequestID) UnmarshalJSON(b []byte) error {
	id.intError = ffjson.Unmarshal(b, &id.intValue)
	id.floatError = ffjson.Unmarshal(b, &id.floatValue)
	id.stringError = ffjson.Unmarshal(b, &id.stringValue)

	return nil
}

func (id *RequestID) MarshalJSON() ([]byte, error) {
	if id.intError == nil {
		return ffjson.Marshal(id.intValue)
	} else if id.floatError == nil {
		return ffjson.Marshal(id.floatValue)
	} else {
		return ffjson.Marshal(id.stringValue)
	}
}

// Int returns the ID as an integer value.
// An error is returned if the ID can't be treated as an int.
func (id *RequestID) Int() (int, error) {
	return id.intValue, id.intError
}

// Float32 returns the ID as a float value.
// An error is returned if the ID can't be treated as an float.
func (id *RequestID) Float32() (float32, error) {
	return id.floatValue, id.floatError
}

// String returns the ID as a string value.
// An error is returned if the ID can't be treated as an string.
func (id *RequestID) String() (string, error) {
	return id.stringValue, id.stringError
}

// Response defines a JSON RPC response from the spec
// http://www.jsonrpc.org/specification#response_object
type Response struct {
	JSONRPC string     `json:"jsonrpc"`
	Result  RawMessage `json:"result,omitempty"`
	Error   *Error     `json:"error,omitempty"`
	ID      *RequestID `json:"id"`
}

const (
	// Version defines the version of the JSON RPC implementation
	Version string = "2.0"

	// ContentType defines the content type to be served.
	ContentType string = "application/json; charset=utf-8"
)
