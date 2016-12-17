// Copyright (C) 2016 Space Monkey, Inc.

package infectuous

import (
	"bytes"
	"testing"

	"sm/space/rand"
)

func TestBasicOperation(t *testing.T) {
	const block = 1024 * 1024
	const total, required = 40, 20

	code, err := NewFecCode(required, total)
	if err != nil {
		t.Fatalf("failed to create new fec code: %s", err)
	}

	// seed the initial data
	data := make([]byte, required*block)
	for i := range data {
		data[i] = byte(i)
	}

	// encode it and store to outputs
	var outputs = make(map[int][]byte)
	store := func(idx, total int, data []byte) {
		outputs[idx] = append([]byte(nil), data...)
	}
	err = code.Encode(data[:], store)
	if err != nil {
		t.Fatalf("encode failed: %s", err)
	}

	// pick required of the total shares randomly
	var shares [required]Share
	for i, idx := range rand.Perm(total)[:required] {
		shares[i].Number = idx
		shares[i].Data = outputs[idx]
	}

	got := make([]byte, required*block)
	record := func(idx, total int, data []byte) {
		copy(got[idx*block:], data)
	}
	err = code.Decode(shares[:], record)
	if err != nil {
		t.Fatalf("decode failed: %s", err)
	}

	if !bytes.Equal(got, data) {
		t.Fatalf("did not match")
	}
}

func BenchmarkEncode(b *testing.B) {
	const block = 1024 * 1024
	const total, required = 40, 20

	code, err := NewFecCode(required, total)
	if err != nil {
		b.Fatalf("failed to create new fec code: %s", err)
	}

	// seed the initial data
	data := make([]byte, required*block)
	for i := range data {
		data[i] = byte(i)
	}
	store := func(idx, total int, data []byte) {}

	b.SetBytes(block * required)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		code.Encode(data, store)
	}
}

func BenchmarkDecode(b *testing.B) {
	const block = 4096
	const total, required = 40, 20

	code, err := NewFecCode(required, total)
	if err != nil {
		b.Fatalf("failed to create new fec code: %s", err)
	}

	// seed the initial data
	data := make([]byte, required*block)
	for i := range data {
		data[i] = byte(i)
	}
	shares := make([]Share, total)
	store := func(idx, total int, data []byte) {
		shares[idx].Number = idx
		shares[idx].Data = append(shares[idx].Data, data...)
	}
	err = code.Encode(data, store)
	if err != nil {
		b.Fatalf("failed to encode: %s", err)
	}

	dec_shares := shares[total-required:]

	b.SetBytes(block * required)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		code.Decode(dec_shares, nil)
	}
}
