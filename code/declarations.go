// Package code provides a tree-sitter backed parser for source code files.
//
// It extracts structural declarations (types, functions, classes, interfaces,
// etc.) from source code and maps them to mq's unified Document model:
//
//   - H1 headings: type/class/struct/interface/enum/trait declarations
//   - H2 headings: function/method declarations
//   - H3 headings: import blocks, constant/variable declarations
//   - Sections: each declaration body with signature as preview
//   - CodeBlocks: one block for the entire file
//   - ReadableText: full source text
//
// Uses odvcencio/gotreesitter (pure Go, no CGo) for AST parsing.
package code

// headingLevel defines what heading level a node type maps to.
// 0 means "unwrap this container and check its children" (for decorators, exports).
// 1 = H1 (types/classes), 2 = H2 (functions/methods), 3 = H3 (imports/constants).
type headingLevel = int

const (
	levelUnwrap headingLevel = 0
	levelType   headingLevel = 1
	levelFunc   headingLevel = 2
	levelImport headingLevel = 3
)

// declarationTypes maps language names to their declaration node types
// and the heading level each should produce.
var declarationTypes = map[string]map[string]headingLevel{
	"go": {
		"type_declaration":     levelType,
		"function_declaration": levelFunc,
		"method_declaration":   levelFunc,
		"import_declaration":   levelImport,
		"package_clause":       levelImport,
	},
	"python": {
		"class_definition":      levelType,
		"function_definition":   levelFunc,
		"decorated_definition":  levelUnwrap,
		"import_statement":      levelImport,
		"import_from_statement": levelImport,
	},
	"typescript": {
		"class_declaration":      levelType,
		"interface_declaration":  levelType,
		"enum_declaration":       levelType,
		"type_alias_declaration": levelType,
		"function_declaration":   levelFunc,
		"method_definition":      levelFunc,
		"export_statement":       levelUnwrap,
		"lexical_declaration":    levelFunc,
		"import_statement":       levelImport,
	},
	"tsx": {
		"class_declaration":      levelType,
		"interface_declaration":  levelType,
		"enum_declaration":       levelType,
		"type_alias_declaration": levelType,
		"function_declaration":   levelFunc,
		"method_definition":      levelFunc,
		"export_statement":       levelUnwrap,
		"lexical_declaration":    levelFunc,
		"import_statement":       levelImport,
	},
	"javascript": {
		"class_declaration":    levelType,
		"function_declaration": levelFunc,
		"method_definition":    levelFunc,
		"export_statement":     levelUnwrap,
		"lexical_declaration":  levelFunc,
		"import_statement":     levelImport,
	},
	"rust": {
		"struct_item":     levelType,
		"trait_item":      levelType,
		"enum_item":       levelType,
		"impl_item":       levelType,
		"function_item":   levelFunc,
		"use_declaration": levelImport,
		"mod_item":        levelImport,
	},
	"java": {
		"class_declaration":       levelType,
		"interface_declaration":   levelType,
		"enum_declaration":        levelType,
		"method_declaration":      levelFunc,
		"constructor_declaration": levelFunc,
		"import_declaration":      levelImport,
		"package_declaration":     levelImport,
	},
	"c": {
		"struct_specifier":    levelType,
		"enum_specifier":      levelType,
		"union_specifier":     levelType,
		"function_definition": levelFunc,
		"declaration":         levelFunc,
		"preproc_include":     levelImport,
	},
	"cpp": {
		"struct_specifier":     levelType,
		"class_specifier":      levelType,
		"enum_specifier":       levelType,
		"union_specifier":      levelType,
		"namespace_definition": levelType,
		"function_definition":  levelFunc,
		"declaration":          levelFunc,
		"preproc_include":      levelImport,
	},
	"ruby": {
		"class":            levelType,
		"module":           levelType,
		"method":           levelFunc,
		"singleton_method": levelFunc,
	},
	"swift": {
		"class_declaration":    levelType,
		"struct_declaration":   levelType,
		"enum_declaration":     levelType,
		"protocol_declaration": levelType,
		"function_declaration": levelFunc,
		"import_declaration":   levelImport,
	},
	"kotlin": {
		"class_declaration":     levelType,
		"object_declaration":    levelType,
		"interface_declaration": levelType,
		"function_declaration":  levelFunc,
		"import_header":         levelImport,
		"package_header":        levelImport,
	},
	"php": {
		"class_declaration":     levelType,
		"interface_declaration": levelType,
		"trait_declaration":     levelType,
		"function_definition":   levelFunc,
		"method_declaration":    levelFunc,
	},
	"scala": {
		"class_definition":    levelType,
		"object_definition":   levelType,
		"trait_definition":    levelType,
		"function_definition": levelFunc,
		"val_definition":      levelFunc,
		"import_declaration":  levelImport,
		"package_clause":      levelImport,
	},
	"zig": {
		"TopLevelDecl":  levelFunc,
		"FnProto":       levelFunc,
		"ContainerDecl": levelType,
	},
	"lua": {
		"function_declaration":          levelFunc,
		"function_definition_statement": levelFunc,
		"local_function":                levelFunc,
	},
	"elixir": {
		"call": levelFunc, // defmodule, def, defp are all calls in elixir grammar
	},
	"haskell": {
		"function":   levelFunc,
		"type_alias": levelType,
		"newtype":    levelType,
		"adt":        levelType,
		"class":      levelType,
		"instance":   levelType,
		"import":     levelImport,
	},
	"sql": {
		"create_table_statement":    levelType,
		"create_view_statement":     levelType,
		"create_function_statement": levelFunc,
		"select_statement":          levelFunc,
	},
	"css": {
		"rule_set":         levelFunc,
		"media_statement":  levelType,
		"import_statement": levelImport,
	},
	"toml": {
		"table":               levelType,
		"table_array_element": levelType,
		"pair":                levelFunc,
	},
}

// getDeclarationLevel returns the heading level for a node type in a given language.
// Returns -1 if the node type is not a known declaration.
func getDeclarationLevel(langName, nodeType string) headingLevel {
	if types, ok := declarationTypes[langName]; ok {
		if level, ok := types[nodeType]; ok {
			return level
		}
	}
	return -1
}
