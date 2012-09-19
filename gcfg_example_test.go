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

func ExampleParseString_subsections() {
	cfgStr := `; Comment line
[profile "A"]
color = white

[profile "B"]
color = black
`
	cfg := struct {
		Profile map[string]*struct {
			Color string
		}
	}{}
	err := ParseString(&cfg, cfgStr)
	if err != nil {
		log.Fatalf("Failed to parse INI data: %s", err)
	}
	fmt.Printf("%s %s\n", cfg.Profile["A"].Color, cfg.Profile["B"].Color)
	// Output: white black
}
