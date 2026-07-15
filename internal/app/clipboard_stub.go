//go:build !windows

package app

import "errors"

func readSystemClipboard() (string, uint, error) {
	return "", 0, errors.New("クリップボード変換はWindows版でのみ利用できます")
}

func writeSystemClipboard(string) error {
	return errors.New("クリップボード変換はWindows版でのみ利用できます")
}
