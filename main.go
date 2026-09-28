package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kodechain/kottg/compiler"
)

const VERSION = "1.0.0"

func main() {
	// Define command-line flags
	inputFile := flag.String("input", "", "Input kottg source file (.kg)")
	outputFile := flag.String("output", "", "Output bytecode file (default: <input>.bin)")
	disassemble := flag.Bool("disasm", false, "Show disassembled bytecode")
	showAST := flag.Bool("ast", false, "Show Abstract Syntax Tree")
	verbose := flag.Bool("verbose", false, "Verbose output  ")
	version := flag.Bool("version", false, "Show version    ")
	jsonOutput := flag.Bool("json", false, "Output bytecode metadata as JSON")

	flag.Parse()

	// Show version
	if *version {
		fmt.Printf("kottg compiler version %s\n", VERSION)
		fmt.Println("KodeChain On-chain Transaction Typing & Governance Language")
		return
	}

	// Validate input file
	if *inputFile == "" {
		fmt.Println("Error: No input file specified")
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		fmt.Println("\nExample:")
		fmt.Println("  kottg -input contracts/StakingQuery.kg")
		fmt.Println("  kottg -input contracts/StakingQuery.kg -disasm")
		fmt.Println("  kottg -input contracts/StakingQuery.kg -output build/StakingQuery.bin")
		os.Exit(1)
	}

	// Check if input file exists
	if _, err := os.Stat(*inputFile); os.IsNotExist(err) {
		fmt.Printf("Error: Input file '%s' does not exist\n", *inputFile)
		os.Exit(1)
	}

	// Read source file
	source, err := ioutil.ReadFile(*inputFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("═══════════════════════════════════════════\n")
		fmt.Printf("KOTTG COMPILER v%s\n", VERSION)
		fmt.Printf("═══════════════════════════════════════════\n\n")
		fmt.Printf("Input file: %s\n", *inputFile)
		fmt.Printf("Source size: %d bytes\n\n", len(source))
	}

	// Phase 1: Lexical Analysis
	if *verbose {
		fmt.Println("Phase 1: Lexical Analysis...")
	}

	lexer := compiler.NewLexer(string(source))

	// Phase 2: Parsing
	if *verbose {
		fmt.Println("Phase 2: Parsing...")
	}

	parser := compiler.NewParser(lexer)
	program := parser.ParseProgram()

	// Check for parsing errors
	if len(parser.Errors()) > 0 {
		fmt.Println("\n❌ PARSING ERRORS:")
		for _, errMsg := range parser.Errors() {
			fmt.Printf("  - %s\n", errMsg)
		}
		os.Exit(1)
	}

	if *verbose {
		fmt.Println("✅ Parsing successful!")
		fmt.Printf("   Contracts: %d\n", len(program.Contracts))
		fmt.Printf("   Top-level structs: %d\n\n", len(program.Structs))
	}

	// Show AST if requested
	if *showAST {
		fmt.Println("\n═══════════════════════════════════════════")
		fmt.Println("ABSTRACT SYNTAX TREE")
		fmt.Println("═══════════════════════════════════════════")
		printAST(program)
	}

	// Phase 3: Semantic Analysis
	if *verbose {
		fmt.Println("Phase 3: Semantic Analysis...")
	}

	semanticAnalyzer := compiler.NewSemanticAnalyzer()
	if !semanticAnalyzer.Analyze(program) {
		fmt.Println("\n" + semanticAnalyzer.PrettyPrint())
		os.Exit(1)
	}

	if *verbose {
		fmt.Println("✅ Semantic analysis successful!")
		fmt.Println()
	}

	// Phase 4: Bytecode Generation
	if *verbose {
		fmt.Println("Phase 4: Bytecode Generation...")
	}

	bytecodeGen := compiler.NewBytecodeGenerator()
	bytecode, err := bytecodeGen.Generate(program)
	if err != nil {
		fmt.Printf("\n❌ BYTECODE GENERATION ERROR: %v\n", err)
		if len(bytecodeGen.Errors()) > 0 {
			for _, errMsg := range bytecodeGen.Errors() {
				fmt.Printf("  - %s\n", errMsg)
			}
		}
		os.Exit(1)
	}

	if *verbose {
		fmt.Println("✅ Bytecode generation successful!")
		fmt.Printf("   Bytecode size: %d bytes\n\n", len(bytecode))
	}

	// Show disassembly if requested
	if *disassemble {
		fmt.Println("\n═══════════════════════════════════════════")
		fmt.Println("DISASSEMBLY")
		fmt.Println("═══════════════════════════════════════════")
		printDisassembly(bytecodeGen)
	}

	// Output JSON metadata if requested
	if *jsonOutput {
		jsonData, err := bytecodeGen.ToJSON()
		if err != nil {
			fmt.Printf("Error generating JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\n═══════════════════════════════════════════")
		fmt.Println("BYTECODE METADATA (JSON)")
		fmt.Println("═══════════════════════════════════════════")
		fmt.Println(jsonData)
	}

	// Determine output file
	outFile := *outputFile
	if outFile == "" {
		ext := filepath.Ext(*inputFile)
		outFile = strings.TrimSuffix(*inputFile, ext) + ".bin"
	}

	// Write bytecode to file
	err = ioutil.WriteFile(outFile, bytecode, 0644)
	if err != nil {
		fmt.Printf("Error writing output file: %v\n", err)
		os.Exit(1)
	}

	// Success message
	fmt.Println("\n═══════════════════════════════════════════")
	fmt.Println("✅ COMPILATION SUCCESSFUL")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("\nContract compiled successfully!\n")
	fmt.Printf("Output: %s\n", outFile)
	fmt.Printf("Bytecode: 0x%s\n", hex.EncodeToString(bytecode[:min(32, len(bytecode))]))
	if len(bytecode) > 32 {
		fmt.Printf("... (%d bytes total)\n", len(bytecode))
	}
	fmt.Println("\nReady for deployment to KodeChain!")
}

// printAST prints a formatted view of the AST
func printAST(program *compiler.Program) {
	// Print top-level structs
	if len(program.Structs) > 0 {
		fmt.Println("Top-Level Structs:")
		for _, s := range program.Structs {
			fmt.Printf("  struct %s {\n", s.Name)
			for _, field := range s.Fields {
				fmt.Printf("    %s: %s\n", field.Name, field.Type.TypeName())
			}
			fmt.Println("  }")
		}
		fmt.Println()
	}

	// Print contracts
	for _, contract := range program.Contracts {
		fmt.Printf("Contract: %s\n", contract.Name)

		// State variables
		if contract.State != nil && len(contract.State.Variables) > 0 {
			fmt.Println("  State Variables:")
			for _, v := range contract.State.Variables {
				fmt.Printf("    %s: %s\n", v.Name, v.Type.TypeName())
			}
		}

		// Events
		if len(contract.Events) > 0 {
			fmt.Println("  Events:")
			for _, e := range contract.Events {
				fmt.Printf("    event %s {\n", e.Name)
				for _, field := range e.Fields {
					indexed := ""
					if field.Indexed {
						indexed = " indexed"
					}
					fmt.Printf("      %s: %s%s\n", field.Name, field.Type.TypeName(), indexed)
				}
				fmt.Println("    }")
			}
		}

		// Functions
		if len(contract.Functions) > 0 {
			fmt.Println("  Functions:")
			for _, f := range contract.Functions {
				vis := f.Visibility
				if vis == "" {
					vis = "priv"
				}
				viewStr := ""
				if f.IsView {
					viewStr = " view"
				}
				retType := "void"
				if f.ReturnType != nil {
					retType = f.ReturnType.TypeName()
				}

				modifiers := ""
				if len(f.Modifiers) > 0 {
					modifiers = fmt.Sprintf(" [%s]", strings.Join(f.Modifiers, ", "))
				}

				fmt.Printf("    %s%s fn %s(...) -> %s%s\n", vis, viewStr, f.Name, retType, modifiers)
			}
		}

		fmt.Println()
	}
}

// printDisassembly prints the disassembled bytecode
func printDisassembly(gen *compiler.BytecodeGenerator) {
	fmt.Print(gen.Disassemble())
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
