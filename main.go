package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

func GetGoodEntries(dir string) ([]os.DirEntry, []os.DirEntry) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	jpg := make([]os.DirEntry, 0)
	raw := make([]os.DirEntry, 0)
	for _, entry := range entries {
		switch filepath.Ext(entry.Name()) {
		case ".jpg":
			jpg = append(jpg, entry)
		case ".tif", ".dng":
			raw = append(raw, entry)
		default:
			continue
		}
	}
	return jpg, raw
}

func better_sort(a, b os.DirEntry) int {
	re := regexp.MustCompile(`\d+`)
	a_int, err := strconv.Atoi(re.FindString(a.Name()))
	b_int, err_2 := strconv.Atoi(re.FindString(b.Name()))
	if err != nil || err_2 != nil {
		panic(err)
	}
	return a_int - b_int
}

func dry(width int, pattern string, entries []os.DirEntry) {
	for i, entry := range entries {
		fmt.Printf("%s -> %s\n", entry.Name(), fmt.Sprintf("%s-%0*d%s", pattern, width, i+1, filepath.Ext(entry.Name())))
	}
}

func revert(recent, original []os.DirEntry) {
	for i, entry := range original { // Assuming they have the exact same length
		err := os.Rename(recent[i].Name(), entry.Name())
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s -> %s\n", recent[i].Name(), entry.Name())
	}
}

func real(width int, pattern string, entries []os.DirEntry) []os.DirEntry {
	new_list := make([]os.DirEntry, 0, len(entries))
	for i, entry := range entries {
		rename := fmt.Sprintf("%s-%0*d%s", pattern, width, i+1, filepath.Ext(entry.Name()))
		err := os.Rename(entry.Name(), rename)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s -> %s\n", entry.Name(), rename)
		info, err := os.Stat(rename)
		if err != nil {
			panic(err)
		}
		new_list = append(new_list, fs.FileInfoToDirEntry(info))
	}
	return new_list
}

func main() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	jpg, raw := GetGoodEntries(dir)
	my_cmp := better_sort
	slices.SortFunc(raw, my_cmp)
	slices.SortFunc(jpg, my_cmp)
	recent_jpg := slices.Clone(jpg)
	recent_raw := slices.Clone(raw)

	scanner := bufio.NewScanner(os.Stdin)
	help_msg := fmt.Sprintln("\t dry - Print a dry run of new file names\n" +
		"\t help, h - Display this help message\n" +
		"\t list, ls - List the current found photos names\n" +
		"\t quit, q - Exit the program (an empty line also exits)\n" +
		"\t revert - Undo the last run, restoring the original file names\n" +
		"\t run - Rename the files\n" +
		"\t set <pattern> - Set the file name pattern (default: current directory name)")
	pattern := filepath.Base(dir)
	if len(raw)+len(jpg) == 0 {
		fmt.Println("No Files Found")
		return
	}
	width_raw := len(strconv.Itoa(len(raw)))
	width_jpg := len(strconv.Itoa(len(jpg)))
	fmt.Print(help_msg)

loop:
	for scanner.Scan() {
		text := strings.Split(scanner.Text(), " ")
		switch text[0] {
		case "quit", "", "q":
			break loop
		case "help", "h":
			fmt.Print(help_msg)
			continue
		case "run":
			recent_raw = real(width_raw, pattern, recent_raw)
			recent_jpg = real(width_jpg, pattern, recent_jpg)
			fmt.Println("DONE!")
			continue
		case "list", "ls":
			for _, entry := range recent_raw {
				fmt.Println(entry.Name())
			}
			for _, entry := range recent_jpg {
				fmt.Println(entry.Name())
			}
			continue
		case "dry":
			dry(width_raw, pattern, raw)
			dry(width_jpg, pattern, jpg)
			continue
		case "set":
			if len(text) < 2 {
				fmt.Println("No pattern provided")
				break
			}
			pattern = text[1]
			fmt.Printf("Pattern set to [%s]\n", pattern)
		case "revert":
			if slices.Equal(recent_jpg, jpg) && slices.Equal(recent_raw, raw) {
				fmt.Println("Nothing to revert")
				break
			}
			revert(recent_jpg, jpg)
			revert(recent_raw, raw)
			recent_raw = raw
			recent_jpg = jpg
		default:
			fmt.Println("No command found: ", text)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error: ", err)
	}
}
