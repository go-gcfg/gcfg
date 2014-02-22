package gcfg

import (
	"fmt"
	"strings"
)

type gbool bool

var gboolValues = map[string]gbool{
	"true": true, "yes": true, "on": true, "1": true,
	"false": false, "no": false, "off": false, "0": false}

func (b *gbool) UnmarshalText(text []byte) error {
	s := strings.ToLower(string(text))
	v, ok := gboolValues[s]
	if !ok {
		return fmt.Errorf("failed to parse %#q as bool", s)
	}
	*b = gbool(v)
	return nil
}

var _ textUnmarshaler = new(gbool)
