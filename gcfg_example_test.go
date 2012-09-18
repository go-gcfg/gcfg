package gcfg

import (
	"fmt"
	"log"
)

func ExampleParseString() {
	cfgStr := `; Comment line
[section]
name=value # comment`
	cfg := struct {
		Section struct {
			Name string
		}
	}{}
	err := ParseString(&cfg, cfgStr)
	if err != nil {
		log.Fatalf("Failed to parse INI data: %s", err)
	}
	fmt.Println(cfg.Section.Name)
	// Output: value
}
