package main

import (
	"fmt"
	"os"

	"github.com/paalgyula/summit/pkg/summit/tools"
	"github.com/spf13/cobra"
)

func main() {

	var rootCmd = &cobra.Command{
		Use:   "datagen",
		Short: "A CLI tool for generating files",
		Long:  "A CLI tool to generate/re-generate required assets for the WoW server",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Help()
				os.Exit(0)
			}
			// fmt.Printf("Generating file with parameters: outfile=%s, type=%s, url=%s, generator=%s\n", outfile, fileType, url, generator)
			// Your code here
		},
	}

	rootCmd.AddCommand(&cobra.Command{
		Use:   "dbc",
		Short: "Convert DBC files to go source", // I know..
	})

	rootCmd.AddCommand(opcodeGenCommand())
	rootCmd.AddCommand(headerConvertCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func opcodeGenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "opcodes",
		Short: "generates opcodes .go source",
		Run: func(cmd *cobra.Command, args []string) {
			outfile := cmd.Flag("outfile").Value.String()
			packageName := cmd.Flag("package").Value.String()

			r, err := tools.Fetch(tools.OpcodeHeaderURL)
			if err != nil {
				fmt.Printf("cannot fetch opcode source file: %s\n", err.Error())
				os.Exit(1)
			}

			opcodes, err := tools.ParseOpcodes(r)
			if err != nil {
				fmt.Printf("cannot parse opcode source file: %s\n", err.Error())
				os.Exit(1)
			}

			outFile, err := os.Create(outfile)
			if err != nil {
				fmt.Printf("cannot create output file: %s\n", err.Error())
				os.Exit(1)
			}

			err = tools.WriteOpcodeSource(packageName, opcodes, outFile)
			if err != nil {
				fmt.Printf("cannot write opcode source file: %s\n", err.Error())
				os.Exit(1)
			}

			fmt.Println("opcode source file written to:", outfile)
		},
	}

	cmd.Flags().StringP("outfile", "o", "opcodes.go", "output file")
	cmd.Flags().StringP("package", "p", "wow", "go package name")

	return cmd
}

func headerConvertCommand() *cobra.Command {
	var useEndField bool

	cmd := &cobra.Command{
		Use:   "header",
		Short: "convert C++ enums to go source",
		Run: func(cmd *cobra.Command, args []string) {
			outfile := cmd.Flag("outfile").Value.String()
			infile := cmd.Flag("infile").Value.String()
			fromUrl := cmd.Flag("fromUrl").Value.String()
			enumName := cmd.Flag("enumName").Value.String()
			packageName := cmd.Flag("packageName").Value.String()

			err := convertHeader(
				packageName, infile, outfile,
				fromUrl, enumName,
				useEndField)

			if err != nil {
				fmt.Println(err)
			}
		},
	}

	cmd.Flags().StringP("packageName", "p", "", "output package name")
	cmd.Flags().StringP("outfile", "o", "", "output file")
	cmd.Flags().StringP("infile", "i", "", "input file")
	cmd.Flags().StringP("fromUrl", "u", "", "input from url")
	cmd.Flags().StringP("enumName", "e", "", "use single enum with name. eg.: UpdateField")
	cmd.Flags().BoolVar(&useEndField, "useEndField", false, "use end field in enums")

	return cmd
}

func convertHeader(packageName, inFile, outFile, fromUrl, enumName string, useEndField bool) (err error) {
	if packageName == "" {
		return fmt.Errorf("package name is required (--packageName)")
	}

	if inFile == "" {
		return fmt.Errorf("infile is required (--infile)")
	}

	f, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("cannot open input: %w", err)
	}

	enums := tools.ParseHeaderFile(f)

	var out = os.Stdout
	if outFile != "" {
		var err error
		out, err = os.Create(outFile)
		if err != nil {
			return fmt.Errorf("cannot open output: %w", err)
		}
		defer out.Close()
	}

	var opts []tools.WriterOption
	if useEndField {
		opts = append(opts, tools.WithEndField(true))
	}

	if enumName != "" {
		opts = append(opts, tools.WithSingleEnum(enumName))
	}

	tools.WriteGoSource(packageName, enums, out, opts...)

	return nil
}
