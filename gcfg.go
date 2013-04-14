// Package gcfg reads "INI-style" text-based configuration files with
// "name=value" pairs grouped into sections (gcfg files).
//
// This package is still a work in progress; see the sections below for planned
// changes.
//
// Syntax
//
// The syntax is based on that used by git config:
// http://git-scm.com/docs/git-config#_syntax .
// There are some (planned) differences compared to the git config format:
//  - improve data portability:
//    - must be encoded in UTF-8 (for now) and must not contain the 0 byte
//    - include and "path" type is not supported
//      (path type may be implementable as a user-defined type)
//  - disallow potentially ambiguous or misleading definitions:
//    - `[sec.sub]` format is not allowed (deprecated in gitconfig)
//    - `[sec ""]` is not allowed
//      - use `[sec]` for section name "sec" and empty subsection name
//    - (planned) within a single file, definitions must be contiguous for each:
//      - section: '[secA]' -> '[secB]' -> '[secA]' is an error
//      - subsection: '[sec "A"]' -> '[sec "B"]' -> '[sec "A"]' is an error
//      - multivalued variable: 'multi=a' -> 'other=x' -> 'multi=b' is an error
//
// Data structure
//
// The functions in this package read values into a user-defined struct.
// Each section corresponds to a struct field in the config struct, and each
// variable in a section corresponds to a data field in the section struct.
// The name of the field must match the name of the section or variable,
// ignoring case.
// Hyphens in section and variable names correspond to underscores in field
// names.
// For sections with subsections, the corresponding field in config must be a
// map, rather than a struct, with string keys and pointer-to-struct values.
// Values for subsection variables are stored in the map with the subsection
// name used as the map key.
// (Note that unlike section and variable names, subsection names are case
// sensitive.)
// When using a map, and there is a section with the same section name but
// without a subsection name, its values are stored with the empty string used
// as the key.
//
// The functions in this package panic if config is not a pointer to a struct,
// or when a field is not of a suitable type (either a struct or a map with
// string keys and pointer-to-struct values).
//
// Parsing of values
//
// The section structs in the config struct may contain arbitrary types.
// For string fields, the (unquoted and unescaped) value string is assigned to
// the field.
// For bool fields, the field is set to true if the value is "true", "yes", "on"
// or "1", and set to false if the value is "false", "no", "off" or "0",
// ignoring case.
// Unnamed slice types (those whose type description starts with `[]`) are
// handled as multi-value variables (each value is added to the slice).
// For all other types, fmt.Sscanf with the verb "%v" is used to parse the value
// string and set it to the field.
// This means that built-in Go types are parseable using the standard format,
// and any user-defined type is parseable if it implements the fmt.Scanner
// interface.
// Note that the value is considered invalid unless fmt.Scanner fully consumes
// the value string without error.
//
// TODO
//
// The following is a list of changes under consideration:
//  - syntax
//    - define valid section and variable names
//      (consider supporting names starting with non-letter Unicode characters)
//    - reconsider valid escape sequences
//      (gitconfig doesn't support \r in value, \t in subsection name, etc.)
//    - define handling of "implicit value" for types other than bool
//    - consider handling of numeric values (decimal only by default?)
//    - complete syntax documentation
//  - reading
//    - define internal representation structure
//    - support multiple inputs (readers, strings, files)
//    - support declaring encoding (?)
//    - support automatic dereferencing of pointer fields (?)
//    - support varying fields sets for subsections (?)
//  - simplify parsing for user-defined types
//    - export scanEnum
//      - should use longest match (?)
//      - support matching on unique prefix (?)
//    - declare an interface that is easier to implement than fmt.Scanner
//  - writing gcfg files
//  - error handling
//    - report position of extra characters in value
//    - make error context accessible programmatically?
//    - limit input size?
//  - move TODOs to issue tracker (eventually)
//
package gcfg
