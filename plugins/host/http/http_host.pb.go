//go:build !wasip1

// Code generated by protoc-gen-go-plugin. DO NOT EDIT.
// versions:
// 	protoc-gen-go-plugin v0.1.0
// 	protoc               v5.29.3
// source: host/http/http.proto

package http

import (
	context "context"
	wasm "github.com/knqyf263/go-plugin/wasm"
	wazero "github.com/tetratelabs/wazero"
	api "github.com/tetratelabs/wazero/api"
)

const (
	i32 = api.ValueTypeI32
	i64 = api.ValueTypeI64
)

type _httpService struct {
	HttpService
}

// Instantiate a Go-defined module named "env" that exports host functions.
func Instantiate(ctx context.Context, r wazero.Runtime, hostFunctions HttpService) error {
	envBuilder := r.NewHostModuleBuilder("env")
	h := _httpService{hostFunctions}

	envBuilder.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h._Get), []api.ValueType{i32, i32}, []api.ValueType{i64}).
		WithParameterNames("offset", "size").
		Export("get")

	envBuilder.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h._Post), []api.ValueType{i32, i32}, []api.ValueType{i64}).
		WithParameterNames("offset", "size").
		Export("post")

	envBuilder.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h._Put), []api.ValueType{i32, i32}, []api.ValueType{i64}).
		WithParameterNames("offset", "size").
		Export("put")

	envBuilder.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h._Delete), []api.ValueType{i32, i32}, []api.ValueType{i64}).
		WithParameterNames("offset", "size").
		Export("delete")

	envBuilder.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h._Patch), []api.ValueType{i32, i32}, []api.ValueType{i64}).
		WithParameterNames("offset", "size").
		Export("patch")

	envBuilder.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h._Head), []api.ValueType{i32, i32}, []api.ValueType{i64}).
		WithParameterNames("offset", "size").
		Export("head")

	envBuilder.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h._Options), []api.ValueType{i32, i32}, []api.ValueType{i64}).
		WithParameterNames("offset", "size").
		Export("options")

	_, err := envBuilder.Instantiate(ctx)
	return err
}

func (h _httpService) _Get(ctx context.Context, m api.Module, stack []uint64) {
	offset, size := uint32(stack[0]), uint32(stack[1])
	buf, err := wasm.ReadMemory(m.Memory(), offset, size)
	if err != nil {
		panic(err)
	}
	request := new(HttpRequest)
	err = request.UnmarshalVT(buf)
	if err != nil {
		panic(err)
	}
	resp, err := h.Get(ctx, request)
	if err != nil {
		panic(err)
	}
	buf, err = resp.MarshalVT()
	if err != nil {
		panic(err)
	}
	ptr, err := wasm.WriteMemory(ctx, m, buf)
	if err != nil {
		panic(err)
	}
	ptrLen := (ptr << uint64(32)) | uint64(len(buf))
	stack[0] = ptrLen
}

func (h _httpService) _Post(ctx context.Context, m api.Module, stack []uint64) {
	offset, size := uint32(stack[0]), uint32(stack[1])
	buf, err := wasm.ReadMemory(m.Memory(), offset, size)
	if err != nil {
		panic(err)
	}
	request := new(HttpRequest)
	err = request.UnmarshalVT(buf)
	if err != nil {
		panic(err)
	}
	resp, err := h.Post(ctx, request)
	if err != nil {
		panic(err)
	}
	buf, err = resp.MarshalVT()
	if err != nil {
		panic(err)
	}
	ptr, err := wasm.WriteMemory(ctx, m, buf)
	if err != nil {
		panic(err)
	}
	ptrLen := (ptr << uint64(32)) | uint64(len(buf))
	stack[0] = ptrLen
}

func (h _httpService) _Put(ctx context.Context, m api.Module, stack []uint64) {
	offset, size := uint32(stack[0]), uint32(stack[1])
	buf, err := wasm.ReadMemory(m.Memory(), offset, size)
	if err != nil {
		panic(err)
	}
	request := new(HttpRequest)
	err = request.UnmarshalVT(buf)
	if err != nil {
		panic(err)
	}
	resp, err := h.Put(ctx, request)
	if err != nil {
		panic(err)
	}
	buf, err = resp.MarshalVT()
	if err != nil {
		panic(err)
	}
	ptr, err := wasm.WriteMemory(ctx, m, buf)
	if err != nil {
		panic(err)
	}
	ptrLen := (ptr << uint64(32)) | uint64(len(buf))
	stack[0] = ptrLen
}

func (h _httpService) _Delete(ctx context.Context, m api.Module, stack []uint64) {
	offset, size := uint32(stack[0]), uint32(stack[1])
	buf, err := wasm.ReadMemory(m.Memory(), offset, size)
	if err != nil {
		panic(err)
	}
	request := new(HttpRequest)
	err = request.UnmarshalVT(buf)
	if err != nil {
		panic(err)
	}
	resp, err := h.Delete(ctx, request)
	if err != nil {
		panic(err)
	}
	buf, err = resp.MarshalVT()
	if err != nil {
		panic(err)
	}
	ptr, err := wasm.WriteMemory(ctx, m, buf)
	if err != nil {
		panic(err)
	}
	ptrLen := (ptr << uint64(32)) | uint64(len(buf))
	stack[0] = ptrLen
}

func (h _httpService) _Patch(ctx context.Context, m api.Module, stack []uint64) {
	offset, size := uint32(stack[0]), uint32(stack[1])
	buf, err := wasm.ReadMemory(m.Memory(), offset, size)
	if err != nil {
		panic(err)
	}
	request := new(HttpRequest)
	err = request.UnmarshalVT(buf)
	if err != nil {
		panic(err)
	}
	resp, err := h.Patch(ctx, request)
	if err != nil {
		panic(err)
	}
	buf, err = resp.MarshalVT()
	if err != nil {
		panic(err)
	}
	ptr, err := wasm.WriteMemory(ctx, m, buf)
	if err != nil {
		panic(err)
	}
	ptrLen := (ptr << uint64(32)) | uint64(len(buf))
	stack[0] = ptrLen
}

func (h _httpService) _Head(ctx context.Context, m api.Module, stack []uint64) {
	offset, size := uint32(stack[0]), uint32(stack[1])
	buf, err := wasm.ReadMemory(m.Memory(), offset, size)
	if err != nil {
		panic(err)
	}
	request := new(HttpRequest)
	err = request.UnmarshalVT(buf)
	if err != nil {
		panic(err)
	}
	resp, err := h.Head(ctx, request)
	if err != nil {
		panic(err)
	}
	buf, err = resp.MarshalVT()
	if err != nil {
		panic(err)
	}
	ptr, err := wasm.WriteMemory(ctx, m, buf)
	if err != nil {
		panic(err)
	}
	ptrLen := (ptr << uint64(32)) | uint64(len(buf))
	stack[0] = ptrLen
}

func (h _httpService) _Options(ctx context.Context, m api.Module, stack []uint64) {
	offset, size := uint32(stack[0]), uint32(stack[1])
	buf, err := wasm.ReadMemory(m.Memory(), offset, size)
	if err != nil {
		panic(err)
	}
	request := new(HttpRequest)
	err = request.UnmarshalVT(buf)
	if err != nil {
		panic(err)
	}
	resp, err := h.Options(ctx, request)
	if err != nil {
		panic(err)
	}
	buf, err = resp.MarshalVT()
	if err != nil {
		panic(err)
	}
	ptr, err := wasm.WriteMemory(ctx, m, buf)
	if err != nil {
		panic(err)
	}
	ptrLen := (ptr << uint64(32)) | uint64(len(buf))
	stack[0] = ptrLen
}
