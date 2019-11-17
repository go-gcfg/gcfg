package gcfg_test

import (
	"fmt"
	"log"
)

import "gopkg.in/gcfg.v1"

func ExampleReadStringInto() {
	cfgStr := `; Comment line
[section]
name=value # comment`
	cfg := struct {
		Section struct {
			Name string
		}
	}{}
	err := gcfg.ReadStringInto(&cfg, cfgStr)
	if err != nil {
		log.Fatalf("Failed to parse gcfg data: %s", err)
	}
	fmt.Println(cfg.Section.Name)
	// Output: value
}

func ExampleReadStringInto_bool() {
	cfgStr := `; Comment line
[section]
switch=on`
	cfg := struct {
		Section struct {
			Switch bool
		}
	}{}
	err := gcfg.ReadStringInto(&cfg, cfgStr)
	if err != nil {
		log.Fatalf("Failed to parse gcfg data: %s", err)
	}
	fmt.Println(cfg.Section.Switch)
	// Output: true
}

func ExampleReadStringInto_hyphens() {
	cfgStr := `; Comment line
[section-name]
variable-name=value # comment`
	cfg := struct {
		Section_Name struct {
			Variable_Name string
		}
	}{}
	err := gcfg.ReadStringInto(&cfg, cfgStr)
	if err != nil {
		log.Fatalf("Failed to parse gcfg data: %s", err)
	}
	fmt.Println(cfg.Section_Name.Variable_Name)
	// Output: value
}

func ExampleReadStringInto_tags() {
	cfgStr := `; Comment line
[tags]
tag1=value1 # a tag
tag2=value2 # a second tag`
	cfg := struct {
		Tags map[string]string
	}{}
	err := gcfg.ReadStringInto(&cfg, cfgStr)
	if err != nil {
		log.Fatalf("Failed to parse gcfg data: %s", err)
	}
	for k, v := range cfg.Tags {
		fmt.Printf("%s = %s\n", k, v)
	}
	// Output: tag1 = value1
	// tag2 = value2
}

func ExampleReadStringInto_optional_section() {
	cfgStr := `; Comment line
        [Section]
          Name=value`
	cfg := struct {
		Optional *struct {
			Name string
		}
		Section *struct {
			Name string
		}
	}{}
	err := gcfg.ReadStringInto(&cfg, cfgStr)
	if err != nil {
		log.Fatalf("Failed to parse gcfg data: %s", err)
	}
	if cfg.Optional == nil {
		fmt.Println("optional not given")
	} else {
		fmt.Printf("Optional.Name=%s\n", cfg.Optional.Name)
	}
	if cfg.Section == nil {
		fmt.Println("section not given")
	} else {
		fmt.Printf("Section.Name = %s\n", cfg.Section.Name)
	}
	// Output: optional not given
	// Section.Name = value
}

func ExampleReadStringInto_subsections() {
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
	err := gcfg.ReadStringInto(&cfg, cfgStr)
	if err != nil {
		log.Fatalf("Failed to parse gcfg data: %s", err)
	}
	fmt.Printf("%s %s\n", cfg.Profile["A"].Color, cfg.Profile["B"].Color)
	// Output: white black
}

func ExampleReadStringInto_multivalue() {
	cfgStr := `; Comment line
[section]
multi=value1
multi=value2`
	cfg := struct {
		Section struct {
			Multi []string
		}
	}{}
	err := gcfg.ReadStringInto(&cfg, cfgStr)
	if err != nil {
		log.Fatalf("Failed to parse gcfg data: %s", err)
	}
	fmt.Println(cfg.Section.Multi)
	// Output: [value1 value2]
}

func ExampleReadStringInto_unicode() {
	cfgStr := `; Comment line
[甲]
乙=丙 # comment`
	cfg := struct {
		X甲 struct {
			X乙 string
		}
	}{}
	err := gcfg.ReadStringInto(&cfg, cfgStr)
	if err != nil {
		log.Fatalf("Failed to parse gcfg data: %s", err)
	}
	fmt.Println(cfg.X甲.X乙)
	// Output: 丙
}
