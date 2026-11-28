package main

import (
	"io"

	ndhost "github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/stream"
)

func get(in input) (*ndhost.HTTPStreamResponse, error) {
	return ndhost.HTTPSendStream(ndhost.HTTPStreamRequest{Method: "GET", URL: in.URL, TimeoutMs: in.TimeoutMs})
}

// download copies a response body into a stored file inside Navidrome, with stream_copy.
//
//go:wasmexport download
func download() int32 {
	return run(func(in input) output {
		resp, err := get(in)
		if err != nil {
			return fail(err)
		}
		defer resp.Body.Close()
		f, err := ndhost.StorageCreate(in.Path)
		if err != nil {
			return fail(err)
		}
		n, err := stream.Copy(f.(*stream.Stream), resp.Body.(*stream.Stream))
		if err != nil {
			return fail(err)
		}
		return output{Count: n, Status: resp.StatusCode, Error: errString(f.Close())}
	})
}

//go:wasmexport fetch
func fetch() int32 {
	return run(func(in input) output {
		resp, err := get(in)
		if err != nil {
			return fail(err)
		}
		data, err := readString(resp.Body)
		return output{Data: data, Status: resp.StatusCode, Error: errString(err)}
	})
}

//go:wasmexport fetch_return
func fetchReturn() int32 {
	return run(func(in input) output {
		resp, err := get(in)
		if err != nil {
			return fail(err)
		}
		return output{Handle: resp.Body.(*stream.Stream).Handle(), Status: resp.StatusCode}
	})
}

// fetch_leak reads one byte of the response body and returns without closing it.
//
//go:wasmexport fetch_leak
func fetchLeak() int32 {
	return run(func(in input) output {
		resp, err := get(in)
		if err != nil {
			return fail(err)
		}
		n, err := resp.Body.Read(make([]byte, 1))
		return output{Count: int64(n), Error: errString(err)}
	})
}

// upload sends a stored file as the request body, then tries to read the consumed body handle again.
//
//go:wasmexport upload
func upload() int32 {
	return run(func(in input) output {
		body, err := ndhost.StorageOpen(in.Path)
		if err != nil {
			return fail(err)
		}
		resp, err := ndhost.HTTPSendStream(ndhost.HTTPStreamRequest{Method: "PUT", URL: in.URL, Headers: in.Headers, Body: body})
		_, reuseErr := body.Read(make([]byte, 1))
		if err != nil {
			out := fail(err)
			out.Data = errString(reuseErr)
			return out
		}
		defer resp.Body.Close()
		return output{Status: resp.StatusCode, Data: errString(reuseErr)}
	})
}

//go:wasmexport open_return
func openReturn() int32 {
	return run(func(in input) output {
		rc, err := ndhost.StorageOpen(in.Path)
		if err != nil {
			return fail(err)
		}
		return output{Handle: rc.(*stream.Stream).Handle()}
	})
}

//go:wasmexport open_path
func openPath() int32 {
	return run(func(in input) output {
		rc, err := ndhost.StorageOpen(in.Path)
		if err != nil {
			return fail(err)
		}
		data, err := readString(rc)
		return output{Data: data, Error: errString(err)}
	})
}

//go:wasmexport create_path
func createPath() int32 {
	return run(func(in input) output {
		wc, err := ndhost.StorageCreate(in.Path)
		if err != nil {
			return fail(err)
		}
		if _, err := io.WriteString(wc, in.Text); err != nil {
			return fail(err)
		}
		return output{Error: errString(wc.Close())}
	})
}

//go:wasmexport create_read
func createRead() int32 {
	return run(func(in input) output {
		wc, err := ndhost.StorageCreate(in.Path)
		if err != nil {
			return fail(err)
		}
		defer wc.Close()
		_, err = wc.(io.Reader).Read(make([]byte, 1))
		return fail(err)
	})
}
