package cmd

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/twpayne/go-vfs"
)

func getKernelInfo(fs vfs.FS) (map[string]string, error) {
	const procKernel = "/proc/sys/kernel"
	files := []string{"version", "ostype", "osrelease"}

	stat, err := fs.Stat(procKernel)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("expected %q to be a directory", procKernel)
	}

	res := map[string]string{}
	for _, file := range files {
		p := filepath.Join(procKernel, file)
		f, err := fs.Open(p)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, err
		}
		res[file] = strings.TrimSpace(string(data))
	}

	return res, nil
}

// getOSRelease returns the operating system identification data as defined by
// https://www.freedesktop.org/software/systemd/man/os-release.html.
func getOSRelease(fs vfs.FS) (map[string]string, error) {
	for _, filename := range []string{"/usr/lib/os-release", "/etc/os-release"} {
		f, err := fs.Open(filename)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		defer f.Close()
		m, err := parseOSRelease(f)
		if err != nil {
			return nil, err
		}
		return m, nil
	}
	return nil, os.ErrNotExist
}

// maybeUnquote removes quotation marks around s.
func maybeUnquote(s string) string {
	// Try to unquote.
	if s, err := strconv.Unquote(s); err == nil {
		return s
	}
	// Otherwise return s, unchanged.
	return s
}

// parseOSRelease parses operating system identification data from r as defined
// by https://www.freedesktop.org/software/systemd/man/os-release.html.
func parseOSRelease(r io.Reader) (map[string]string, error) {
	result := make(map[string]string)
	s := bufio.NewScanner(r)
	for s.Scan() {
		// trim all leading whitespace, but not necessarily trailing whitespace
		token := strings.TrimLeftFunc(s.Text(), unicode.IsSpace)
		// if the line is empty or starts with #, skip
		if len(token) == 0 || token[0] == '#' {
			continue
		}
		fields := strings.SplitN(token, "=", 2)
		if len(fields) != 2 {
			return nil, fmt.Errorf("cannot parse %q", token)
		}
		key := fields[0]
		value := maybeUnquote(fields[1])
		result[key] = value
	}
	return result, s.Err()
}
