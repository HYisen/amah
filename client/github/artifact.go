package github

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
)

func CloseLogError(c io.Closer) {
	if err := c.Close(); err != nil {
		slog.Warn("close fail", "err", err)
	}
}

func attachmentName(header http.Header) (string, error) {
	value := header.Get("Content-Disposition")
	prefix := `attachment; filename="`
	suffix := `"`
	if !strings.HasPrefix(value, prefix) || !strings.HasSuffix(value, suffix) {
		return "", fmt.Errorf("bad filename header value %s", value)
	}
	return value[len(prefix) : len(value)-len(suffix)], nil
}

func tempNamePattern(name string) string {
	ext := path.Ext(name)
	return fmt.Sprintf("%s_tmp_*%s", name[:len(name)-len(ext)], ext)
}

// downloadToTempFile does req and save the response body to a temporary file.
// It is the caller's responsibility to remove the file when it is no longer needed.
// Use http.NewRequestWithContext to create req with ctx control.
// The implementation can support a speed more than 500 MiB/s.
// ref https://github.com/hyisen/how-to-download
func downloadToTempFile(req *http.Request) (tempName string, err error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer CloseLogError(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("not OK status %s", resp.Status)
	}

	name, err := attachmentName(resp.Header)
	if err != nil {
		return "", err
	}
	// I don't unzip on the air to reduce memory footprint. So not tmpfs.
	tempFile, err := os.CreateTemp(".", tempNamePattern(name))
	if err != nil {
		return "", err
	}

	if _, err = tempFile.ReadFrom(resp.Body); err != nil {
		// Even if we can't save, tried to clean up the created temp file.
		return "", errors.Join(err, os.Remove(tempFile.Name()))
	}
	return tempFile.Name(), nil
}

func depressSingleContentZip(sourcePath, destinationPath string) error {
	zr, err := zip.OpenReader(sourcePath)
	if err != nil {
		return err
	}
	defer CloseLogError(zr)

	if len(zr.File) != 1 {
		return fmt.Errorf("not one file count %d", len(zr.File))
	}
	src, err := zr.File[0].Open()
	if err != nil {
		return err
	}
	defer CloseLogError(src)

	dst, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer CloseLogError(dst)

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}

// DownloadArtifact downloads the artifact that could be fetched from req,
// unzip it and output to destinationPath. Use NewDownloadArtifactRequest to create a normal req.
func DownloadArtifact(req *http.Request, destinationPath string) (err error) {
	tempName, err := downloadToTempFile(req)
	if err != nil {
		return fmt.Errorf("download zip: %v", err)
	}
	defer func() {
		err = errors.Join(err, os.Remove(tempName))
	}()

	if err := depressSingleContentZip(tempName, destinationPath); err != nil {
		return fmt.Errorf("depress zip: %v", err)
	}
	return err
}

type DownloadArtifactRequest struct {
	Owner       string
	Repo        string
	ArtifactID  int
	AccessToken string
}

// NewDownloadArtifactRequest create the *http.Request with ctx and req.
// ref https://docs.github.com/en/rest/actions/artifacts?apiVersion=2022-11-28#download-an-artifact
func NewDownloadArtifactRequest(ctx context.Context, req DownloadArtifactRequest) (*http.Request, error) {
	url := fmt.Sprintf(
		"https://api.github.com/repos/%v/%v/actions/artifacts/%d/zip",
		req.Owner,
		req.Repo,
		req.ArtifactID,
	)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		// Maybe some string in req is injected, user's fault.
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", "Bearer "+req.AccessToken)
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	return request, nil
}
