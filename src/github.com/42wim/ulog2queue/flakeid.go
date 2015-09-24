package main

import (
	"encoding/base64"
	"encoding/binary"
	"github.com/davidnarayan/go-flake"
)

var (
	flaker   *flake.Flake
	flakeB64 []byte
)

func init() {
	var err error
	// 9 bytes for base64 encoding, factor of 3 (for now until upgrade to go1.5)
	flakeB64 = make([]byte, 9)
	flaker, err = flake.New()
	if err != nil {
		log.Fatal(err)
	}
}

func getFlakeID() string {
	binary.BigEndian.PutUint64(flakeB64, uint64(flaker.NextId()))
	return base64.URLEncoding.EncodeToString(flakeB64)
}
