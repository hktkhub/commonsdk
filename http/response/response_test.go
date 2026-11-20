package response_test

import (
	"errors"
	"testing"

	"github.com/hktkhub/commonsdk/http/response"
)

type fakeCode int64

type codedError struct {
	code fakeCode
	msg  string
}

func (e codedError) Error() string { return e.msg }

func (e codedError) Code() fakeCode { return e.code }

func TestFromErrorExtractsCode(t *testing.T) {
	env := response.FromError(codedError{code: 30043, msg: "quota exceeded"})

	if env.Code != 30043 {
		t.Fatalf("code = %d, want 30043", env.Code)
	}
	if env.Msg != "quota exceeded" {
		t.Fatalf("msg = %q, want quota exceeded", env.Msg)
	}
	if env.Data != nil {
		t.Fatalf("data = %v, want nil", env.Data)
	}
}

func TestFromErrorPlainError(t *testing.T) {
	env := response.FromError(errors.New("boom"))
	if env.Code != 0 {
		t.Fatalf("code = %d, want 0", env.Code)
	}
	if env.Msg != "boom" {
		t.Fatalf("msg = %q, want boom", env.Msg)
	}
}

func TestFromErrorWithCode(t *testing.T) {
	env := response.FromErrorWithCode(500, errors.New("boom"))
	if env.Code != 500 {
		t.Fatalf("code = %d, want 500", env.Code)
	}
}

func TestFromErrorNil(t *testing.T) {
	env := response.FromError(nil)
	if env.Code != 0 || env.Msg != "ok" {
		t.Fatalf("envelope = %+v", env)
	}
}
