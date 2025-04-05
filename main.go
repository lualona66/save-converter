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
	"strings"

	"github.com/sqweek/dialog"
)

const (
	maxSaveFileSize = 256 * 1024      // 256KB
	maxRomFileSize  = 64 * 1024 * 1024 // 64MB
	fullmempakSize  = 128 * 1024      // .mpk 32KB x 4 Controller Paks
	mempakSize 	= 32 * 1024
)

var allowedSaveExtensions = map[string]bool{
	".eep":    true,
	".eeprom": true,
	".fla":    true,
	".flash":  true,
	".mpk":    true,
	".pak":    true,
	".ram":    true,
	".sra":    true,
}

var allowedRomExtensions = map[string]bool{
	".z64": true,
	".n64": true,
	".v64": true,
}

var romFormats = map[string]string{
	"z64": "\x80\x37\x12\x40", // Big-endian
	"n64": "\x40\x12\x37\x80", // Little-endian
	"v64": "\x37\x80\x40\x12", // Byte-swapped
}

var aresSaveFormatMap = map[string]string{
	".eeprom": ".eep",
	".flash":  ".fla",
	".pak":    ".mpk",
	".ram":    ".sra",
}

var nonConversionExtensions = map[string]bool{
	".eep":    true,
	".eeprom": true,
	".flash":  true,
	".mpk":    true,
	".pak":    true,
	".ram":    true,
}

var ConversionExtensions = map[string]bool{
	".fla":    true,
	".sra":    true,
}

func printUsage() {
	fmt.Println("N64 emulator save converter: > Gopher64")
	fmt.Println("\nValid save files:", strings.Join(getKeys(allowedSaveExtensions), " "))
	fmt.Println("Valid N64 rom files:", strings.Join(getKeys(allowedRomExtensions), " "))
	fmt.Println("\nUsage:")
	switch runtime.GOOS {
		case "windows":
			fmt.Println("    save-converter.exe <save_file> <N64_rom_file>")
		default:
			fmt.Println("    ./save-converter <save_file> <N64_rom_file>") // General case for other OSes
	}
}

// getKeys helper function to extract keys from a map for usage printing
func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// processRom extracts Rom name and computes hash
func processRom(romFile string) (cleanTitle string, hashHex string, err error) {
	romHandle, err := os.Open(romFile)
	if err != nil {
		return "", "", fmt.Errorf("Error opening ROM file: %w", err)
	}
	defer romHandle.Close()

	rawHeader := make([]byte, 0x40)
	if _, err := io.ReadFull(romHandle, rawHeader); err != nil {
		return "", "", fmt.Errorf("Error reading ROM header: %w", err)
	}

	romMode, err := detectRomFormat(rawHeader[:4])
	if err != nil {
		return "", "", err
	}

	convHeader := convertHeaderEndianness(rawHeader, romMode)
	cleanTitle = extractCleanTitle(convHeader)
	hashHex, err = computeSHA256(romFile)
	if err != nil {
		return "", "", err
	}

	return cleanTitle, hashHex, nil
}

// checks rom header and assigns correct rom format
func detectRomFormat(magicBytes []byte) (string, error) {
	for mode, magic := range romFormats {
		if string(magicBytes) == magic {
			return mode, nil
		}
	}
	return "", fmt.Errorf("Error unsupported ROM format based on magic bytes: %x", magicBytes)
}

// converts and extracts header for rom name
func convertHeaderEndianness(rawHeader []byte, romMode string) []byte {
	convHeader := make([]byte, len(rawHeader))
	switch romMode {
		case "z64":
			copy(convHeader, rawHeader)
		case "n64":
			for i := 0; i < len(rawHeader); i += 4 {
				binary.BigEndian.PutUint32(convHeader[i:i+4], binary.LittleEndian.Uint32(rawHeader[i:i+4]))
			}
		case "v64":
			for i := 0; i < len(rawHeader); i += 2 {
				binary.BigEndian.PutUint16(convHeader[i:i+2], binary.LittleEndian.Uint16(rawHeader[i:i+2]))
			}
	}
	return convHeader
}

// extracts rom name from converted header then removes special characters and trailing spaces
func extractCleanTitle(convHeader []byte) string {
	titleField := convHeader[0x20 : 0x20+20]
	titleBytes := bytes.TrimRight(titleField, " ")
	title := string(titleBytes)

	var cleanTitleRunes strings.Builder
	for _, r := range title {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			cleanTitleRunes.WriteRune(r)
		}
	}
	return cleanTitleRunes.String()
}

// compute SHA256 hash for rom file and return HEX string
func computeSHA256(romFile string) (string, error) {
	romHashFile, err := os.Open(romFile)
	if err != nil {
		return "", fmt.Errorf("Error opening ROM file for hashing: %w", err)
	}
	defer romHashFile.Close()

	hasher := sha256.New()
	if _, err := io.CopyBuffer(hasher, romHashFile, make([]byte, 4096)); err != nil {
		return "", fmt.Errorf("Error computing SHA256 for ROM file: %w", err)
	}
	hashSum := hasher.Sum(nil)
	return fmt.Sprintf("%X", hashSum), nil
}

// read the save file, processes it in 4-byte chunks (swapping endianness)
func convertSaveFile(src, dst string) error {
	fmt.Printf("\nConverting Data from: %s\n", src)
	inFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("Error opening save file: %w", err)
	}
	defer inFile.Close()

	outFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("Error creating output file: %w", err)
	}
	defer outFile.Close()

	buf := make([]byte, 4)
	for {
		n, err := io.ReadFull(inFile, buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("Error reading save file: %w", err)
		}
		if n != 4 {
			return fmt.Errorf("Error unexpected read size from save file: got %d bytes, expected 4", n)
		}

		binary.LittleEndian.PutUint32(buf, binary.BigEndian.Uint32(buf))

		if _, err := outFile.Write(buf); err != nil {
			return fmt.Errorf("Error writing to output file: %w", err)
		}
	}

	return outFile.Sync()
}

// checks file size and trims or pads as needed then save file. (4x data copy if 32KB save)
func copyFile(src, dst string, targetSize int64) error {
	fmt.Printf("\nCopying Data from: %s\n", src)
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("Error opening source file: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("Error creating destination file: %w", err)
	}
	defer out.Close()

	if targetSize == 0 {
		_, err = io.Copy(out, in)
		if err != nil {
			return fmt.Errorf("Error copying file: %w", err)
		}
		return out.Sync()
	}

	fileData, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("Error reading source file into memory: %w", err)
	}
	fileSize := int64(len(fileData))
	if fileSize == 0 {
		return fmt.Errorf("Error source file is empty")
	}

	// fmt.Println("fileSize before initial processing:", fileSize)


	if fileSize > fullmempakSize {
		fileData = fileData[:fullmempakSize]
		fileSize = fullmempakSize
		fmt.Println("Source file was larger than fullmempakSize, trimmed to 128KB.")
	}


	if fileSize <= mempakSize {
		paddingSize := mempakSize - fileSize
		fmt.Println("File smaller than mempakSize. Padding", paddingSize, "bytes to mempakSize")
		padding := make([]byte, paddingSize) // Zero-filled padding
		paddedFileData := make([]byte, 0, mempakSize)
		paddedFileData = append(paddedFileData, fileData...)
		paddedFileData = append(paddedFileData, padding...)
		fileData = paddedFileData
		fileSize = int64(len(fileData))
	}
	// fmt.Println("fileSize after padding:", fileSize)


	_, err = out.Write(fileData)
	if err != nil {
		return fmt.Errorf("Error writing initial data (trimmed/padded) to destination file: %w", err)
	}


	numCopies := 1 // Start with 1 because we already have base content
	if targetSize > fileSize {
		numCopies += int((targetSize - fileSize) / fileSize)
	}

	bytesToPad := 0
	if targetSize > fileSize {
		bytesToPad = int((targetSize - fileSize) % fileSize)
	}


	// fmt.Println("numCopies", numCopies)
	// fmt.Println("bytesToPad", bytesToPad)


	if numCopies > 1 {
		for i := 1; i < numCopies; i++ {
		_, err = out.Write(fileData)
		if err != nil {
			return fmt.Errorf("Error writing repeated data to destination file: %w", err)
			}
		}
	}


	if bytesToPad > 0 {
		fmt.Println("Padding", bytesToPad, "final bytes")
		extrapadding := make([]byte, bytesToPad)
		_, err = out.Write(extrapadding)
		if err != nil {
			return fmt.Errorf("Error writing final padding data to destination file: %w", err)
		}
	}


	return out.Sync()
}


func validateFile(filePath string, isDir bool, maxSize int64, allowedExtensions map[string]bool, fileType string) error {
	fmt.Println("Validating",fileType, filePath)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("Error accessing %s file '%s': %w", fileType, filePath, err)
	}
	if fileInfo.IsDir() != isDir {
		if isDir {
			return fmt.Errorf("Error: '%s' is not a directory, expected a directory for %s file", filePath, fileType)
		}
		return fmt.Errorf("Error: '%s' is a directory, expected a file for %s file", filePath, fileType)
	}
	if !isDir && fileInfo.Size() > maxSize {
		return fmt.Errorf("Error: %s file '%s' is too large (max size: %dKB)", fileType, filePath, maxSize/1024)
	}
	if !isDir && allowedExtensions != nil {
		ext := filepath.Ext(filePath)
		if !allowedExtensions[ext] {
			allowedExts := strings.Join(getKeys(allowedExtensions), ", ")
			return fmt.Errorf("Errorr: unsupported %s file extension '%s'. Allowed extensions are: %s", fileType, ext, allowedExts)
		}
	}
	return nil
}

// check for arguments, open file picker if none
func main() {
	if len(os.Args) == 1 {
		fmt.Println("No command-line arguments provided. Opening file selector...")
		fmt.Println("\n1: Pick the save file you want to convert.")
		fmt.Println("2: Pick the N64 rom file associated with your save file.")


		var extensions []string
		var romFormats []string

		for ext := range allowedSaveExtensions {
			extensions = append(extensions, strings.TrimPrefix(ext, "."))
		}
		for ext := range allowedRomExtensions {
			romFormats = append(romFormats, strings.TrimPrefix(ext, "."))
		}


		saveFile, err := dialog.File().Title("Select Save File").Filter("Save Files", extensions...).Load()
		if err != nil {
			fmt.Println("\nError selecting save file or operation cancelled.")
			return
		}


		romFile, err := dialog.File().Title("Select N64 ROM File").Filter("ROM Files", romFormats...).Load()
		if err != nil {
			fmt.Println("\nError selecting ROM file or operation cancelled.", err)
			return
		}


		os.Args = []string{"save-converter", saveFile, romFile}
	}

	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			printUsage()
			return
		}
	}

	if len(os.Args) != 3 {
		fmt.Println("Invalid Argument. Expected save file and ROM")
		fmt.Println("  --help for command usage")
		return
	}

	saveFile := os.Args[1]
	if err := validateFile(saveFile, false, maxSaveFileSize, allowedSaveExtensions, "Save:"); err != nil {
		fmt.Println(err)
		return
	}
	saveExt := filepath.Ext(saveFile)

	romFile := os.Args[2]
	if err := validateFile(romFile, false, maxRomFileSize, allowedRomExtensions, "ROM:"); err != nil {
		fmt.Println(err)
		return
	}

	cleanTitle, hashHex, err := processRom(romFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	outputExt := saveExt
	if mappedExt, ok := aresSaveFormatMap[saveExt]; ok {
		outputExt = mappedExt
	}
	outputFile := fmt.Sprintf("%s-%s%s", cleanTitle, hashHex, outputExt)


	if ConversionExtensions[saveExt] {
		if err := convertSaveFile(saveFile, outputFile); err != nil {
			fmt.Println(err)
			return
		}
	} else {
		targetSize := int64(0)
		switch saveExt  {
			case ".mpk":
				targetSize = fullmempakSize
			case ".pak":
				targetSize = fullmempakSize
		}

		if err := copyFile(saveFile, outputFile, targetSize); err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Printf("\nFile converted successfully: %s\n", outputFile)
}
