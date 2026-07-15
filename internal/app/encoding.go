package app

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func decodeText(contents []byte, charset string) (string, error) {
	charset = strings.ToLower(strings.ReplaceAll(charset, "_", "-"))
	if charset == "" || charset == "auto" {
		if utf8.Valid(contents) {
			return string(contents), nil
		}
		charset = "shift-jis"
	}
	if charset == "utf-8" || charset == "utf8" {
		return string(contents), nil
	}
	var decoder *transform.Reader
	switch charset {
	case "shift-jis", "sjis", "cp932", "windows-31j":
		decoder = transform.NewReader(bytes.NewReader(contents), japanese.ShiftJIS.NewDecoder())
	case "euc-jp", "eucjp":
		decoder = transform.NewReader(bytes.NewReader(contents), japanese.EUCJP.NewDecoder())
	case "iso-2022-jp", "iso2022jp", "jis":
		decoder = transform.NewReader(bytes.NewReader(contents), japanese.ISO2022JP.NewDecoder())
	default:
		return "", fmt.Errorf("対応していない文字コードです: %s", charset)
	}
	decoded, err := io.ReadAll(decoder)
	if err != nil {
		return "", fmt.Errorf("%s として文字列を読み込めません: %w", charset, err)
	}
	return string(decoded), nil
}

func formatModifiedAt(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Local().Format("2006/01/02 15:04")
}
