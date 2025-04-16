package ioutil

import (
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

var originTagSeparator = "." // Because I prefer that. It's easy to type manually, comparing to underscore.
var preservedFileExt = ".gz" // It's depended on the actually compress algorithm used. i.e., change to .zstd later.

func preservedFileName(originPath string, tag string) string {
	return originPath + originTagSeparator + tag + preservedFileExt
}

// PreserveFile copied file as a preservation.
// The total preserved file count is limited by maxPreservedFiles.
func PreserveFile(originPath string) (err error) {
	tags, err := existingTags(originPath)
	if err != nil {
		return err
	}

	neoTag, poppedTagOptional := compute(tags)

	if poppedTagOptional != "" {
		if err := os.Remove(preservedFileName(originPath, poppedTagOptional)); err != nil {
			return err
		}
	}

	return CopyCompressed(originPath, preservedFileName(originPath, neoTag))
}

// CopyCompressed reads data from sourcePath and saved the compressed data to destinationPath.
// The compress algorithm is DefaultCompression of gzip.
// I have planned a change to zstd once official supported. [ref](https://github.com/golang/go/issues/62513)
// I prefer zstd because it attracted me as suggested by transparent compression of
// [btrfs](https://btrfs.readthedocs.io/en/latest/Compression.html).
// In the other word, once FileSystem's transparent compression feature is enabled,
// our compress feature here should be disabled.
func CopyCompressed(sourcePath string, destinationPath string) error {
	src, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer CloseLogError(src)

	dst, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer func() {
		// Writer not closable is an Error as data is at risk. Is it flushed and persisted?
		err = errors.Join(err, dst.Close())
	}()

	writer := gzip.NewWriter(dst)
	if _, err = src.WriteTo(writer); err != nil {
		return err
	}
	return writer.Close()
}

// I can tolerate a hand of preserved files, and it shall fit for more than one day.
// Moreover, as I sampled a 9.0% disk space usage after compressed a real log file,
// such amount of preserved files would result a half filesize to origin, seems good.
var maxPreservedFiles = 5

func compute(tags []string) (nextTag string, poppedTagOptional string) {
	if len(tags) == 0 {
		// Start from 1, preserve 0 as an alias of the original.
		return "1", ""
	}

	var nums []int
	for _, tag := range tags {
		num, err := strconv.Atoi(tag)
		if err != nil {
			continue
		}
		nums = append(nums, num)
	}
	slices.Sort(nums)
	if len(nums) >= maxPreservedFiles {
		poppedTagOptional = strconv.Itoa(nums[0])
	}
	return strconv.Itoa(nums[len(nums)-1] + 1), poppedTagOptional
}

func existingTags(originPath string) ([]string, error) {
	dir, err := os.Open(filepath.Dir(originPath))
	if err != nil {
		return nil, err
	}
	defer CloseLogError(dir)

	names, err := dir.Readdirnames(0)
	if err != nil {
		return nil, err
	}

	var ret []string
	originName := filepath.Base(originPath)
	for _, name := range names {
		before, found := strings.CutSuffix(name, preservedFileExt)
		if !found {
			continue
		}
		after, found := strings.CutPrefix(before, originName+originTagSeparator)
		if !found {
			continue
		}
		ret = append(ret, after)
	}
	return ret, nil
}
