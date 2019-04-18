package xpytest

import (
	"fmt"
	"io/ioutil"

	"github.com/golang/protobuf/proto"

	xpytest_proto "github.com/chainer/xpytest/proto"
)

// LoadHintFile loads hint information from a hint file.
func LoadHintFile(file string) (*xpytest_proto.HintFile, error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read hint file: %s", err)
	}
	h := &xpytest_proto.HintFile{}
	if err := proto.UnmarshalText(string(buf), h); err != nil {
		return nil, fmt.Errorf("failed to parse hint file: %s", err)
	}
	return h, nil
}
