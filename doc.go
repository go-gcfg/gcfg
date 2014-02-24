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
//  - internationalization
//    - section and variable names can contain unicode letters, unicode digits
//      (as defined in http://golang.org/ref/spec#Characters ) and hyphens
//      (U+002D), starting with a unicode letter
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
// The mapping of each section or variable name to fields is done either based
// on the "gcfg" struct tag or by matching the name of the section or variable,
// ignoring case. In the latter case, hyphens '-' in section and variable names
// correspond to underscores '_' in field names.
// Fields must be exported; to use a section or variable name starting with a
// letter that is neither upper- or lower-case, prefix the field name with 'X'.
// (See https://code.google.com/p/go/issues/detail?id=5763#c4 .)
//
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
// The section structs in the config struct may contain single-valued or
// multi-valued variables. Multi-valued variables must be of an unnamed slice
// type (that is, a type starting with `[]`); for these, each individual value
// is parsed and added to the slice.
//
// Single-valued variables are handled based on the type as follows. For string
// fields, the value string is assigned to the field, after unquoting and
// unescaping as needed.
//
// For bool fields, the field is set to true if the value is "true", "yes", "on"
// or "1", and set to false if the value is "false", "no", "off" or "0",
// ignoring case. In addition, for single-valued bool fields, the equals sign
// and the value can be omitted; in such case the value is set to true.
//
// For types implementing the encoding.TextUnmarshaler interface, the
// UnmarshalText method is used to set the value. Implementing this method is
// the recommended way for parsing user-defined types.
//
// Predefined integer types [u]int(|8|16|32|64) are parsed as decimal, using
// fmt.Sscanf with the "%d" verb. (This is to prevent unintuitively handling
// zero-padded numbers as octal.) All other types are parsed using fmt.Sscanf
// with the "%v" verb. (Note that this includes types with [u]int* as the
// underlying type, such as os.FileMode. Implementing UnmarshalText is
// recommended in case parsing as octal is undesirable.)
//
// TODO
//
// The following is a list of changes under consideration:
//  - syntax
//    - reconsider valid escape sequences
//      (gitconfig doesn't support \r in value, \t in subsection name, etc.)
//    - self-contained syntax documentation
//  - reading / parsing gcfg files
//    - define internal representation structure
//    - support multiple inputs (readers, strings, files)
//    - support declaring encoding (?)
//    - support varying fields sets for subsections (?)
//  - parsing / setting values
//    - define handling of "implicit value" for types other than bool
//    - support automatic allocation and dereferencing for pointer fields
//      - e.g. allow using *big.Int instead of big.Int
//  - writing gcfg files
//  - error handling
//    - report position of extra characters in value
//    - make error context accessible programmatically?
//    - limit input size?
//  - move TODOs to issue tracker (eventually)
//
package gcfg
