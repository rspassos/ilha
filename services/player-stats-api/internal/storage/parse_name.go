package storage

import (
	"strings"
	"unicode/utf8"
)

var playerNameCharMap = [16][16]string{
	{"•", "", "", "", "", "•", "", "", "", "", "", "", ">", ">", "•", "•"},
	{"[", "]", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "•", "-", "-", "-"},
	{" ", "!", "\"", "#", "$", "%", "&", "'", "(", ")", "*", "+", ",", "-", ".", "/"},
	{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", ":", ";", "<", "=", ">", "?"},
	{"@", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O"},
	{"P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z", "[", "\\", "]", "^", "_"},
	{"'", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O"},
	{"P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z", "{", "|", "}", "~", " "},
	{"(", "=", ")", ".", ".", ".", ".", ".", ".", ".", ".", ".", ">", ">", "•", "•"},
	{"[", "]", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "•", "-", "-", "-"},
	{" ", "!", "\"", "#", "$", "%", "&", "'", "(", ")", "*", "+", ",", "-", ".", "/"},
	{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", ":", ";", "<", "=", ">", "?"},
	{"@", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O"},
	{"P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z", "[", "\\", "]", "^", "_"},
	{"`", "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o"},
	{"p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z", "{", "|", "}", "~", " "},
}

func parsePlayerName(name string) string {
	if name == "" {
		return ""
	}

	var parsed strings.Builder
	parsed.Grow(len(name))

	for index := 0; index < len(name); {
		if mapped, ok := literalUnicodeEscape(name[index:]); ok {
			parsed.WriteString(mapped)
			index += 6
			continue
		}

		current, size := rune(name[index]), 1
		if current >= 0x80 {
			current, size = utf8.DecodeRuneInString(name[index:])
		}
		if mapped, ok := mappedPlayerNameRune(current); ok {
			parsed.WriteString(mapped)
		} else {
			parsed.WriteRune(current)
		}
		index += size
	}

	return parsed.String()
}

func literalUnicodeEscape(value string) (string, bool) {
	if len(value) < 6 || value[0] != '\\' || value[1] != 'u' {
		return "", false
	}

	row, ok := hexNibble(value[4])
	if !ok {
		return "", false
	}
	column, ok := hexNibble(value[5])
	if !ok {
		return "", false
	}

	return playerNameCharMap[row][column], true
}

func mappedPlayerNameRune(value rune) (string, bool) {
	if value < 0x20 || (value >= 0x80 && value <= 0xff) {
		return playerNameCharMap[value>>4][value&0x0f], true
	}
	return "", false
}

func hexNibble(value byte) (rune, bool) {
	switch {
	case value >= '0' && value <= '9':
		return rune(value - '0'), true
	case value >= 'a' && value <= 'f':
		return rune(value-'a') + 10, true
	case value >= 'A' && value <= 'F':
		return rune(value-'A') + 10, true
	default:
		return 0, false
	}
}
