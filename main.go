package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	image          string
	output         string
	files          string
	registryName   string
	repositoryName string
	imageReference string
	username       string
	password       string

	filePaths []string
)

type blobLayer struct {
	blob       []byte
	descriptor ocispec.Descriptor
}

func init() {
	flag.StringVar(&image, "image", "", "oci image address in the format registry/repository:tag (required)")
	flag.StringVar(&username, "username", "", "username for the oci registry/repository (required)")
	flag.StringVar(&password, "password", "", "password for the oci registry/repository (required)")
	flag.StringVar(&files, "files", "", "comma separated file paths (required)")
	flag.StringVar(&output, "output", ".", "path to the output folder (optional)")

	flag.Parse()

	if image == "" || username == "" ||
		password == "" || files == "" {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Parse image into chunks.
	chunks := strings.Split(image, "/")
	subChunks := strings.Split(chunks[2], ":")

	registryName = chunks[0]
	repositoryName = chunks[1] + "/" + subChunks[0]
	imageReference = subChunks[1]

	// Parse files into chunks.
	filePaths = strings.Split(files, ",")
}

func main() {
	push(registryName, repositoryName, imageReference, files, username, password)
	pull(registryName, repositoryName, imageReference, output, username, password)
}

func byteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}
