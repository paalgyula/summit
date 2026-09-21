//nolint:all
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paalgyula/summit/pkg/converter/adt"
	"github.com/paalgyula/summit/pkg/converter/blp"
	"github.com/paalgyula/summit/pkg/converter/m2"
	"github.com/paalgyula/summit/pkg/converter/mpq"
	"github.com/paalgyula/summit/pkg/converter/wmo"
	"github.com/paalgyula/summit/pkg/summit/tools"
	"github.com/paalgyula/summit/pkg/summit/tools/data"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "datagen",
	Short: "A CLI tool for generating files and converting WoW assets",
	Long:  "A CLI tool to generate/re-generate required assets and convert MPQ, BLP, M2, ADT, and WMO assets for web and server",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			_ = cmd.Help()
			os.Exit(0)
		}
	},
}

func main() {
	rootCmd.AddCommand(convertDBC())
	rootCmd.AddCommand(opcodeGenCommand())
	rootCmd.AddCommand(headerConvertCommand())
	rootCmd.AddCommand(convertMPQ())
	rootCmd.AddCommand(convertBLP())
	rootCmd.AddCommand(convertM2())
	rootCmd.AddCommand(convertADT())
	rootCmd.AddCommand(convertWMO())
	rootCmd.AddCommand(migrateCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func convertDBC() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dbc",
		Short: "Convert DBC files to go binary format",
		Run: func(cmd *cobra.Command, args []string) {
			converter := data.NewConverter("dbc")

			if err := converter.CreateSummitBaseData(); err != nil {
				log.Fatal().Err(err).Msg("data conversion failed")
			}
		},
	}

	return cmd
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

	out := os.Stdout

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

func convertMPQ() *cobra.Command {
	var archivePath, extractFile, outDir, format string
	var listFiles, extractAllTextures bool
	var quality float32

	cmd := &cobra.Command{
		Use:   "mpq",
		Short: "Inspect and extract files from Blizzard MPQ archives (with WebP transcoding)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if archivePath == "" {
				return fmt.Errorf("--archive is required")
			}
			arc, err := mpq.Open(archivePath)
			if err != nil {
				return fmt.Errorf("cannot open mpq: %w", err)
			}
			if listFiles {
				files := arc.ListFiles()
				fmt.Printf("MPQ Archive %s contains %d indexed files:\n", archivePath, len(files))
				for _, f := range files {
					fmt.Println(" ", f)
				}
				return nil
			}
			if extractAllTextures {
				files := arc.ListFiles()
				converted := 0
				for _, f := range files {
					if strings.HasSuffix(strings.ToLower(f), ".blp") {
						data, err := arc.ReadFile(f)
						if err != nil {
							continue
						}
						img, err := blp.Decode(bytes.NewReader(data))
						if err != nil {
							continue
						}
						cleanName := strings.ReplaceAll(f, "\\", "/")
						dest := filepath.Join(outDir, strings.TrimSuffix(cleanName, filepath.Ext(cleanName))+".webp")
						_ = os.MkdirAll(filepath.Dir(dest), 0755)
						if err := blp.SaveWebP(img, dest, quality, true); err == nil {
							converted++
						}
					}
				}
				fmt.Printf("Extracted and converted %d textures to WebP in %s\n", converted, outDir)
				return nil
			}
			if extractFile != "" {
				data, err := arc.ReadFile(extractFile)
				if err != nil {
					return fmt.Errorf("failed to extract file: %w", err)
				}
				cleanName := strings.ReplaceAll(extractFile, "\\", "/")
				baseName := filepath.Base(cleanName)

				if strings.ToLower(format) == "webp" && strings.HasSuffix(strings.ToLower(baseName), ".blp") {
					img, err := blp.Decode(bytes.NewReader(data))
					if err != nil {
						return fmt.Errorf("decode extracted blp: %w", err)
					}
					dest := filepath.Join(outDir, strings.TrimSuffix(baseName, filepath.Ext(baseName))+".webp")
					if outDir != "" {
						_ = os.MkdirAll(outDir, 0755)
					}
					if err := blp.SaveWebP(img, dest, quality, true); err != nil {
						return fmt.Errorf("save webp: %w", err)
					}
					fmt.Printf("Extracted & converted %s -> %s (WebP)\n", extractFile, dest)
					return nil
				}

				dest := filepath.Join(outDir, baseName)
				if outDir != "" {
					_ = os.MkdirAll(outDir, 0755)
				}
				if err := os.WriteFile(dest, data, 0644); err != nil {
					return fmt.Errorf("write extracted file: %w", err)
				}
				fmt.Printf("Extracted %s (%d bytes) to %s\n", extractFile, len(data), dest)
				return nil
			}
			return cmd.Help()
		},
	}

	cmd.Flags().StringVarP(&archivePath, "archive", "a", "", "path to MPQ archive")
	cmd.Flags().StringVarP(&extractFile, "extract", "x", "", "file inside MPQ to extract")
	cmd.Flags().StringVarP(&outDir, "out", "o", ".", "output directory for extracted files")
	cmd.Flags().BoolVarP(&listFiles, "list", "l", false, "list all indexed files in MPQ")
	cmd.Flags().BoolVar(&extractAllTextures, "textures-to-webp", false, "batch extract and convert all BLP textures to WebP")
	cmd.Flags().StringVarP(&format, "format", "f", "webp", "transcode BLP to format: webp, png, or raw")
	cmd.Flags().Float32VarP(&quality, "quality", "q", 85.0, "WebP quality (0-100)")

	return cmd
}

func convertBLP() *cobra.Command {
	var inPath, outPath, format string
	var quality float32
	var lossless bool

	cmd := &cobra.Command{
		Use:   "blp",
		Short: "Convert BLP textures to WebP or PNG (single file or directory)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if inPath == "" {
				return fmt.Errorf("--infile is required")
			}

			fi, err := os.Stat(inPath)
			if err != nil {
				return fmt.Errorf("stat input: %w", err)
			}

			convertOne := func(blpFile, destFile string) error {
				img, err := blp.DecodeFile(blpFile)
				if err != nil {
					return fmt.Errorf("decode %s: %w", blpFile, err)
				}
				_ = os.MkdirAll(filepath.Dir(destFile), 0755)
				if strings.ToLower(format) == "png" {
					return blp.SavePNG(img, destFile)
				}
				return blp.SaveWebP(img, destFile, quality, !lossless)
			}

			// Batch directory conversion
			if fi.IsDir() {
				if outPath == "" {
					outPath = inPath
				}
				count := 0
				err := filepath.Walk(inPath, func(path string, info os.FileInfo, err error) error {
					if err != nil || info.IsDir() {
						return err
					}
					if strings.HasSuffix(strings.ToLower(path), ".blp") {
						rel, _ := filepath.Rel(inPath, path)
						targetExt := ".webp"
						if strings.ToLower(format) == "png" {
							targetExt = ".png"
						}
						dest := filepath.Join(outPath, strings.TrimSuffix(rel, filepath.Ext(rel))+targetExt)
						if err := convertOne(path, dest); err != nil {
							fmt.Printf("Warning: failed to convert %s: %v\n", path, err)
						} else {
							count++
						}
					}
					return nil
				})
				if err != nil {
					return err
				}
				fmt.Printf("Batch converted %d textures to %s in %s\n", count, strings.ToUpper(format), outPath)
				return nil
			}

			// Single file conversion
			if outPath == "" {
				targetExt := ".webp"
				if strings.ToLower(format) == "png" {
					targetExt = ".png"
				}
				outPath = strings.TrimSuffix(inPath, filepath.Ext(inPath)) + targetExt
			}

			if err := convertOne(inPath, outPath); err != nil {
				return err
			}
			fmt.Printf("Converted %s -> %s (%s)\n", inPath, outPath, strings.ToUpper(format))
			return nil
		},
	}

	cmd.Flags().StringVarP(&inPath, "infile", "i", "", "input BLP file or directory")
	cmd.Flags().StringVarP(&outPath, "outfile", "o", "", "output file or directory")
	cmd.Flags().StringVarP(&format, "format", "f", "webp", "output format: webp or png")
	cmd.Flags().Float32VarP(&quality, "quality", "q", 85.0, "WebP quality (0-100)")
	cmd.Flags().BoolVar(&lossless, "lossless", false, "use lossless WebP compression")

	return cmd
}

func convertM2() *cobra.Command {
	var inFile, skinFile, outFile string

	cmd := &cobra.Command{
		Use:   "m2",
		Short: "Convert M2 (+ optional .skin) models to glTF 2.0 (.glb)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if inFile == "" {
				return fmt.Errorf("--infile is required")
			}
			if outFile == "" {
				outFile = strings.TrimSuffix(inFile, filepath.Ext(inFile)) + ".glb"
			}
			model, err := m2.Open(inFile)
			if err != nil {
				return fmt.Errorf("open m2: %w", err)
			}
			var skin *m2.Skin
			if skinFile != "" {
				s, err := m2.OpenSkin(skinFile)
				if err != nil {
					return fmt.Errorf("open skin: %w", err)
				}
				skin = s
			} else {
				autoSkin := strings.TrimSuffix(inFile, filepath.Ext(inFile)) + "00.skin"
				if s, err := m2.OpenSkin(autoSkin); err == nil {
					skin = s
					fmt.Printf("Auto-loaded skin file: %s\n", autoSkin)
				}
			}
			doc, err := m2.ConvertToGLTF(model, skin)
			if err != nil {
				return fmt.Errorf("convert m2 to gltf: %w", err)
			}
			if err := doc.SaveGLB(outFile); err != nil {
				return fmt.Errorf("save glb: %w", err)
			}
			fmt.Printf("Converted M2 model %s -> %s (%d vertices, %d bones)\n", inFile, outFile, len(model.Vertices), len(model.Bones))
			return nil
		},
	}

	cmd.Flags().StringVarP(&inFile, "infile", "i", "", "input M2 model file")
	cmd.Flags().StringVarP(&skinFile, "skin", "s", "", "optional .skin file")
	cmd.Flags().StringVarP(&outFile, "outfile", "o", "", "output GLB file")

	return cmd
}

func convertADT() *cobra.Command {
	var inFile, outFile string

	cmd := &cobra.Command{
		Use:   "adt",
		Short: "Convert ADT terrain tiles to glTF 2.0 (.glb)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if inFile == "" {
				return fmt.Errorf("--infile is required")
			}
			if outFile == "" {
				outFile = strings.TrimSuffix(inFile, filepath.Ext(inFile)) + ".glb"
			}
			terrain, err := adt.OpenADT(inFile)
			if err != nil {
				return fmt.Errorf("open adt: %w", err)
			}
			f, err := os.Create(outFile)
			if err != nil {
				return err
			}
			defer f.Close()
			if err := terrain.ExportGLB(f); err != nil {
				return fmt.Errorf("export adt glb: %w", err)
			}
			fmt.Printf("Converted ADT terrain %s -> %s\n", inFile, outFile)
			return nil
		},
	}

	cmd.Flags().StringVarP(&inFile, "infile", "i", "", "input ADT terrain file")
	cmd.Flags().StringVarP(&outFile, "outfile", "o", "", "output GLB file")

	return cmd
}

func convertWMO() *cobra.Command {
	var inFile, outFile string

	cmd := &cobra.Command{
		Use:   "wmo",
		Short: "Convert WMO structures and buildings to glTF 2.0 (.glb)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if inFile == "" {
				return fmt.Errorf("--infile is required")
			}
			if outFile == "" {
				outFile = strings.TrimSuffix(inFile, filepath.Ext(inFile)) + ".glb"
			}
			building, err := wmo.OpenRoot(inFile)
			if err != nil {
				return fmt.Errorf("open wmo: %w", err)
			}
			basePrefix := strings.TrimSuffix(inFile, filepath.Ext(inFile))
			for i := 0; i < int(building.Header.NGroups); i++ {
				grpPath := fmt.Sprintf("%s_%03d.wmo", basePrefix, i)
				gf, err := os.Open(grpPath)
				if err == nil {
					if grp, gErr := wmo.ReadGroup(gf); gErr == nil {
						building.Groups = append(building.Groups, grp)
					}
					gf.Close()
				}
			}
			f, err := os.Create(outFile)
			if err != nil {
				return err
			}
			defer f.Close()
			if err := building.ExportGLB(f); err != nil {
				return fmt.Errorf("export wmo glb: %w", err)
			}
			fmt.Printf("Converted WMO %s -> %s (%d groups loaded)\n", inFile, outFile, len(building.Groups))
			return nil
		},
	}

	cmd.Flags().StringVarP(&inFile, "infile", "i", "", "input Root WMO file")
	cmd.Flags().StringVarP(&outFile, "outfile", "o", "", "output GLB file")

	return cmd
}
