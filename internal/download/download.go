package download

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/xianml/s0cmd/internal/s3"
)

type Downloader struct {
	Parallelism int
	Output      string
	s3Client    *s3.S3Client
	Region      string
}

func (d *Downloader) Download(ctx context.Context, presignedURL string) error {
	// // Get presigned URL
	// presignedURL, err := d.getPresignedURL(ctx, s3Uri)
	// if err != nil {
	// 	return err
	// }

	// Get object size
	size, err := d.getObjectSize(ctx, presignedURL)
	if err != nil {
		return err
	}

	// Create output file
	file, err := os.Create(d.Output)
	if err != nil {
		return errors.Wrap(err, "failed to create target file")
	}
	defer file.Close()

	if err := file.Truncate(size); err != nil {
		return errors.Wrapf(err, "failed to truncate the target file")
	}

	ranges, err := CalculateRange(size, d.Parallelism)
	if err != nil {
		return errors.Wrap(err, "failed to calculate ranges")
	}

	fmt.Println("Downloading...")
	d.Parallelism = len(ranges)
	startTime := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < d.Parallelism; i++ {
		wg.Add(1)
		start_cp := ranges[i][0]
		end_cp := ranges[i][1]
		part_cp := i
		go func(start, end int64, part int) {
			defer wg.Done()
			if err := d.downloadPart(ctx, presignedURL, start, end, file); err != nil {
				fmt.Printf("Error downloading part %d: %v\n", part, err)
			}
		}(start_cp, end_cp, part_cp)
	}
	wg.Wait()

	duration := time.Since(startTime)
	// Convert bytes to megabits (bytes * 8 / 1024 / 1024)
	megabits := float64(size) * 8 / 1024 / 1024
	// Calculate MB/s
	mbps := megabits / duration.Seconds()
	fmt.Printf("Download completed in %.2f seconds\n", duration.Seconds())
	fmt.Printf("Average bandwidth: %.2f Mb/s\n", mbps)
	return nil
}

func (d *Downloader) getObjectSize(ctx context.Context, presignedURL string) (int64, error) {
	cmd := exec.CommandContext(ctx, "curl", "--silent", "--show-error", "--fail", "--head", presignedURL)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to run command: %s, stderr: %s", stringifyCmd(cmd), stderr.String())
	}

	var size int64
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		k, v, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		if strings.ToLower(strings.TrimSpace(k)) == "content-length" {
			var err error
			size, err = strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse size: %s", string(output))
			}
			break
		}
	}
	return size, nil
}

func (d *Downloader) downloadPart(ctx context.Context, presignedURL string, start, end int64, file *os.File) error {
	pr, pw := io.Pipe()
	fmt.Println("Downloading part ", start, "-", end, " ...")
	defer pr.Close()
	defer pw.Close()
	go func() {
		var stderr bytes.Buffer
		defer pw.Close()
		cmd := exec.CommandContext(ctx, "curl", "--silent", "--show-error", "--fail", "--output", "-", "--range", fmt.Sprintf("%d-%d", start, end), presignedURL) // nolint:gosec
		cmd.Stderr = &stderr
		cmd.Stdout = pw
		start_t := time.Now()
		if err := cmd.Run(); err != nil {
			pw.CloseWithError(errors.Wrapf(err, "failed to run command: %s, stderr: %s", stringifyCmd(cmd), stderr.String()))
		}
		fmt.Printf("Part [%v, %v] downloaded in %.2f seconds\n", start, end, time.Since(start_t).Seconds())
	}()

	buf := make([]byte, 64*1024*1024)

	for {
		n, err := pr.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return errors.Wrap(err, "failed to read from pipe")
		}
		_, err = file.WriteAt(buf[:n], start)
		if err != nil {
			return errors.Wrap(err, "failed to write to writer")
		}
		start += int64(n)
	}
	return nil
}

func stringifyCmd(cmd *exec.Cmd) string {
	var b strings.Builder
	b.WriteString(cmd.Path)
	for _, arg := range cmd.Args[1:] {
		b.WriteString(" ")
		b.WriteString(arg)
	}
	return b.String()
}
