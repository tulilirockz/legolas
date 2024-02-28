package main

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
)

const DEPS_LIBPATH = "/lib"

func FileCopy(sourceFile, destFile string) error {
	source, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	if err != nil {
		return err
	}
	return nil
}

func MapVal[T, U any](data []T, f func(T) U) []U {

	res := make([]U, 0, len(data))

	for _, e := range data {
		res = append(res, f(e))
	}

	return res
}
func ErrExit(err error) {
	fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
}

var fViewOnly bool = true

func main() {
	flag.BoolVar(&fViewOnly, "view-only", false, "Only view the binary info")
	flag.Parse()

	if len(os.Args) < 2 {
		ErrExit(errors.New("Failed parsing required argument: FILE RPATH"))
		return
	}

	var selectedFile string = os.Args[1]
	var newRunpath string = os.Args[2]

	read_elf, err := elf.Open(path.Clean(selectedFile))
	if err != nil {
		ErrExit(err)
		return
	}
	defer read_elf.Close()

	needed, err := read_elf.DynString(elf.DT_NEEDED)
	if err != nil {
		ErrExit(err)
		return
	}

	fmt.Println("needed deps:")
	for _, symbol := range needed {
		fmt.Printf("%s\n", symbol) // search for the file in /lib/*
	}

	libdirs, err := os.ReadDir(DEPS_LIBPATH)
	if err != nil {
		ErrExit(err)
		return
	}

	var validPaths []string

	for _, symbol := range needed {
		for _, dirs := range libdirs {
			if !dirs.IsDir() {
				continue
			}

			lib_prefix, err := os.ReadDir(path.Join(DEPS_LIBPATH, dirs.Name()))
			if err != nil {
				ErrExit(err)
				return
			}

			var libraries []string
			for _, prefix := range lib_prefix {
				if prefix.IsDir() {
					continue
				}
				libraries = append(libraries, prefix.Name())
			}

			if slices.Contains(libraries, symbol) {
				validPaths = append(validPaths, path.Join(DEPS_LIBPATH, dirs.Name(), symbol))
			}
		}
	}

	// Find RPATH in dynamic session -> change it.

	rpath, err := read_elf.DynString(elf.DT_RUNPATH)
	if err != nil {
		ErrExit(err)
		return
	}

	fmt.Printf("Library paths:\n%v\n\n", validPaths)
	fmt.Printf("\nRUNPATH library paths:\n%v\n", rpath)

	if fViewOnly {
		os.Exit(0)
		return
	}

	// Get the DynamicSection offset, then add the "DT_RUNPATH" offset to that
	const INVALID_OFFSET = -1

	var dynSectionOffset = INVALID_OFFSET
	for _, section := range read_elf.Sections {
		if section.Name == ".dynstr" {
			dynSectionOffset = int(section.Offset)
		}
	}
	if dynSectionOffset == INVALID_OFFSET {
		ErrExit(errors.New("could not find dynamic section"))
		return
	}

	const NEWFILE_PATH = "newelftest.elf"
	if err := FileCopy(path.Clean(selectedFile), NEWFILE_PATH); err != nil {
		ErrExit(err)
		return
	}

	newFile, err := os.OpenFile(NEWFILE_PATH, os.O_RDWR, 0755)
	if err != nil {
		ErrExit(err)
		return
	}

	newOffsetRunPath, err := newFile.Seek(int64(dynSectionOffset), io.SeekStart)
	if err != nil {
		ErrExit(err)
		return
	}

	if _, err := newFile.WriteString(newRunpath + "\x00"); err != nil {
		ErrExit(err)
		return
	}

	newRunpathVal := elf.Dyn64{
		Tag: int64(elf.DT_RUNPATH),
		Val: uint64(newOffsetRunPath),
	}

	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.LittleEndian, newRunpathVal)
	if err != nil {
		fmt.Println("Error converting struct to bytes:", err)
		return
	}

	var runpathOffset = int64(dynSectionOffset) + int64(elf.DT_RUNPATH)

	if _, err = newFile.Seek(runpathOffset, io.SeekStart); err != nil {
		ErrExit(err)
		return
	}
	written, err := newFile.Write(buf.Bytes())
	if err != nil {
		ErrExit(err)
		return
	}

	fmt.Printf("Patched binary successfully! Created new file: %s, bytes written: %d\n", NEWFILE_PATH, written)
}
