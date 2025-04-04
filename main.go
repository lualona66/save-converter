package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"github.com/sqweek/dialog"
)

const maxSaveFileSize = 256 * 1024 // 256KB
const maxRomFileSize = 64 * 1024 * 1024 // 64MB

func printUsage () {
	fmt.Println("N64emu save converter  # PJ64 > Gopher64 #")
	fmt.Println("\n Valid PJ64 save files: .sra  .fla  .eep  .mpk")
	fmt.Println(" Valid N64 rom files: .z64  .n64  .v64")
	fmt.Println("\nUsage:")
	switch runtime.GOOS {
		case "windows":
			fmt.Println("    save-converter.exe <save_file> <N64_rom_file>")
		case "linux":
			fmt.Println("    ./save-converter <save_file> <N64_rom_file>")
	}
}

// Extract Rom name and compute hash
// 1. Reads the first 64 bytes (header) of the ROM file.
// 2. Determines the ROM format via its magic bytes and converts the header to big‑endian order.
// 3. Extracts and cleans the game title from the converted header.
// 4. Computes the ROM’s SHA256 hash.
// Returns the cleaned title, the SHA256 hash (as a hex string), and any error encountered.
func processRom(romFile string) (cleanTitle string, hashHex string, err error) {
	// Open the ROM file for header processing.
	romHandle, err := os.Open(romFile)
	if err != nil {
		return "", "", fmt.Errorf("error opening ROM file: %w", err)
	}
	defer romHandle.Close()

	// Read the first 64 bytes (0x40) header.
	rawHeader := make([]byte, 0x40)
	n, err := io.ReadFull(romHandle, rawHeader)
	if err != nil {
		return "", "", fmt.Errorf("error reading ROM header: %w", err)
	}
	if n < len(rawHeader) {
		return "", "", fmt.Errorf("ROM header is incomplete")
	}

	// Check magic bytes (first 4 bytes) to determine ROM format.
	// Expected patterns:
	// .z64 (big-endian): 80 37 12 40
	// .n64 (little-endian): 40 12 37 80
	// .v64 (byte‑swapped): 37 80 40 12
	magic := rawHeader[0:4]
	var romMode string
	switch {
		case magic[0] == 0x80 && magic[1] == 0x37 && magic[2] == 0x12 && magic[3] == 0x40:
			romMode = "z64" // Big-endian: no conversion needed.
		case magic[0] == 0x40 && magic[1] == 0x12 && magic[2] == 0x37 && magic[3] == 0x80:
			romMode = "n64" // Little-endian: swap each 4-byte block.
		case magic[0] == 0x37 && magic[1] == 0x80 && magic[2] == 0x40 && magic[3] == 0x12:
			romMode = "v64" // Byte‑swapped: swap every 2 bytes.
		default:
			return "", "", fmt.Errorf("unsupported ROM format based on magic bytes")
	}

	// Convert the raw header so that the title is in the correct big‑endian order.
	convHeader := make([]byte, 0x40)
	switch romMode {
		case "z64":
			copy(convHeader, rawHeader)
		case "n64":
			// For .n64, reverse each 4‑byte block.
			for i := 0; i < 0x40; i += 4 {
				convHeader[i+0] = rawHeader[i+3]
				convHeader[i+1] = rawHeader[i+2]
				convHeader[i+2] = rawHeader[i+1]
				convHeader[i+3] = rawHeader[i+0]
			}
		case "v64":
			// For .v64, swap every 2 bytes.
			for i := 0; i < 0x40; i += 2 {
				convHeader[i+0] = rawHeader[i+1]
				convHeader[i+1] = rawHeader[i+0]
			}
	}

	// The game title is located from offset 0x20 to 0x33 (20 bytes).
	titleField := convHeader[0x20 : 0x20+20]

	// Remove trailing spaces (hex 0x20) used as padding.
	titleBytes := bytes.TrimRight(titleField, " ")
	title := string(titleBytes)

	// Remove any special characters; only allow alphanumerics and spaces.
	cleanTitleRunes := make([]rune, 0, len(title))
	for _, r := range title {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') || r == ' ' {
				cleanTitleRunes = append(cleanTitleRunes, r)
			}
	}
	cleanTitle = string(cleanTitleRunes)

	// Compute SHA256 hash of the ROM file (processing it in its raw form).
	romHashFile, err := os.Open(romFile)
	if err != nil {
		return "", "", fmt.Errorf("error opening ROM file for hashing: %w", err)
	}
	defer romHashFile.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, romHashFile); err != nil {
		return "", "", fmt.Errorf("error computing SHA256 for ROM file: %w", err)
	}
	hashSum := hasher.Sum(nil)
	hashHex = fmt.Sprintf("%X", hashSum)

	return cleanTitle, hashHex, nil
}

// Reads the save file, processes it in 4-byte chunks (swapping endianness),
func convertSaveFile(inPath, outPath string) error {
	inFile, err := os.Open(inPath)
	if err != nil {
		return fmt.Errorf("error opening save file: %w", err)
	}
	defer inFile.Close()

	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer outFile.Close()

	buf := make([]byte, 4096)
	for {
		n, err := inFile.Read(buf[:4]) // Read only 4 bytes at a time for processing
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading save file: %w", err)
		}
		if n < 4 {
			return fmt.Errorf("save file size is not a multiple of 4 bytes")
		}

		// Swap the byte order from BigEndian to LittleEndian in-place.
		binary.LittleEndian.PutUint32(buf[:4], binary.BigEndian.Uint32(buf[:4]))

		if _, err := outFile.Write(buf[:4]); err != nil {
			return fmt.Errorf("error writing to output file: %w", err)
		}
	}

	return outFile.Sync() // Ensure data is written to disk
}


func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening source file: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("error creating destination file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("error copying file: %w", err)
	}

	return out.Sync()
}

func main() {

	// If no arguments are provided, assume the program was double-clicked and open file pickers.
	if len(os.Args) == 1 {
		fmt.Println("No command-line arguments provided. Opening file selectors...")
		fmt.Println("1: Pick the save file you want to convert.")
		fmt.Println("2: Pick the N64 rom file associated with your save file.")

		// Pick the save file.
		saveFile, err := dialog.File().Title("Select Save File").Filter("Save Files", "sra", "fla", "eep", "mpk").Load()
		if err != nil {
			fmt.Println("\nError selecting save file or operation cancelled.")
			return
		}

		// Pick the ROM file
		romFile, err := dialog.File().Title("Select N64 ROM File").Filter("ROM Files", "z64", "n64", "v64").Load()
		if err != nil {
			fmt.Println("\nError selecting ROM file or operation cancelled.", err)
			return
		}

		os.Args = []string{"save-converter.exe", saveFile, romFile}


	}

	// Check for help flags in command-line arguments.
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			printUsage()
			return
		}
	}

	//Check that we have enough arguments (save and ROM files).
	if len(os.Args) != 3 {
		fmt.Println("Invalid Argument")
		fmt.Println("  --help for command usage")
		return
	}

	// Validate the save file.
	saveFile := os.Args[1]
	saveFileInfo, err := os.Stat(saveFile)
	if err != nil || saveFileInfo.IsDir() {
		fmt.Printf("\nError: '%s' is not a valid file path.\n", saveFile)
		return
	}
	// Check the file size against the maximum allowed size
	if saveFileInfo.Size() > maxSaveFileSize {
		fmt.Printf("\nError: Save file '%s' is too large.\n", saveFile)
		return
	}
	allowedSaveExtensions := map[string]bool{
		".sra": true,
		".fla": true,
		".eep": true,
		".mpk": true,
	}
	saveExt := filepath.Ext(saveFile)
	if !allowedSaveExtensions[saveExt] {
		fmt.Printf("\nError: Unsupported save file extension '%s'  Only .sra .fla .eep and .mpk files are allowed.\n", saveExt)
		return
	}

	var outputFile string

	romFile := os.Args[2]
	romFileInfo, err := os.Stat(romFile)
	if err != nil || romFileInfo.IsDir() {
		fmt.Printf("\nError: '%s' is not a valid file path.\n", romFile)
		return
	}
	// Check the file size against the maximum allowed size
	if romFileInfo.Size() > maxRomFileSize {
		fmt.Printf("\nError: Rom file '%s' is too large.\n", romFile)
		return
	}
	allowedRomExtensions := map[string]bool{
		".z64": true,
		".n64": true,
		".v64": true,
	}
	// Input rom file extension check
	romExt := filepath.Ext(romFile)
	if !allowedRomExtensions[romExt] {
		fmt.Printf("\nError: Unsupported ROM file extension '%s'  Only .z64, .n64 and .v64 files are allowed.\n", romExt)
		return
	}

	// Process the ROM to get the cleaned title and SHA256 hash
	cleanTitle, hashHex, err := processRom(romFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	outputFile = fmt.Sprintf("%s-%s%s", cleanTitle, hashHex, saveExt)

	// For .eep and .mpk files, simply copy the file. Otherwise convert save
	nonConversionExtensions := map[string]bool{
		".eep": true,
		".mpk": true,
	}
	if nonConversionExtensions[saveExt] {
		if err := copyFile(saveFile, outputFile); err != nil {
				fmt.Println(err)
				return
			}
	} else {
		if err := convertSaveFile(saveFile, outputFile); err != nil {
			fmt.Println(err)
			return
		}
	}


	fmt.Printf("\nFile converted successfully: %s\n", outputFile)
}
