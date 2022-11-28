// Parts of this code are inspired by the ORAS Project documentation.
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

func push(registryName string, repositoryName string, imageReference string, files string, username string, password string) {
	ctx := context.Background()

	repo, err := remote.NewRepository(fmt.Sprintf("%s/%s", registryName, repositoryName))
	if err != nil {
		panic(err)
	}

	// Setup the client credentials.
	repo.Client = &auth.Client{
		Credential: func(ctx context.Context, reg string) (auth.Credential, error) {
			return auth.Credential{
				Username: username,
				Password: password,
			}, nil
		},
	}

	// The OCI artifact blueprint:
	//   +---------------------------------------------------+
	//   |                                                   |
	//   |                                +----------------+ |
	//   |                             +-->      ....      | |
	//   |            +-------------+  |  +---+ Config +---+ |
	//   | (reference)+-->   ...     +--+                  | |
	//   |            ++ Manifest  ++  |  +----------------+ |
	//   |                             +-->      ...       | |
	//   |                                +---+ Layer  +---+ |
	//   |                                                   |
	//   +------------------+ registry +---------------------+

	// (Blob) Layers creation.
	blobLayers := []blobLayer{}
	blobLayersDescriptors := []ocispec.Descriptor{}

	for _, filePath := range filePaths {
		blob, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalln(fmt.Errorf("failed to read file: %w", err))
		}

		filename := path.Base(filePath)

		blobDescriptor := ocispec.Descriptor{
			MediaType: "application/vnd.oci.image.layer.v1.tar",
			Digest:    digest.FromBytes(blob),
			Size:      int64(len(blob)),
			Annotations: map[string]string{
				ocispec.AnnotationTitle: filename,
			},
		}

		b := blobLayer{
			blob:       blob,
			descriptor: blobDescriptor,
		}

		blobLayers = append(blobLayers, b)
		blobLayersDescriptors = append(blobLayersDescriptors, blobDescriptor)
	}

	// Config creation.
	configBlob := []byte("config")
	configDescriptor := content.NewDescriptorFromBytes("application/vnd.unknown.config.v1+json", configBlob)

	// Manifest creation.
	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Config:    configDescriptor,
		Layers:    blobLayersDescriptors,
		Versioned: specs.Versioned{SchemaVersion: 2},
		Annotations: map[string]string{
			"org.opencontainers.image.created": time.Now().UTC().Format(time.RFC3339),
		},
	}

	manifestBlob, err := json.Marshal(manifest)
	if err != nil {
		log.Fatalln(fmt.Errorf("failed to json encode oci manifest: %w", err))
	}

	manifestDescriptor := content.NewDescriptorFromBytes(ocispec.MediaTypeImageManifest, manifestBlob)

	log.Printf("Pushing blobs to %s/%s:%s\n", registryName, repositoryName, imageReference)

	// Pushing one layer at a time.
	for _, bl := range blobLayers {
		log.Printf("Pushing %s [%s]\n", bl.descriptor.Annotations["org.opencontainers.image.title"],
			byteCountIEC(int64(binary.Size(bl.blob))))

		if err := repo.Push(ctx, bl.descriptor, bytes.NewReader(bl.blob)); err != nil {
			log.Fatalln(fmt.Errorf("failed to blob layer: %w", err))
		}
	}

	// Pushing the config.
	if err := repo.Push(ctx, configDescriptor, bytes.NewReader(configBlob)); err != nil {
		log.Fatalln(fmt.Errorf("failed to push config descriptor: %w", err))
	}

	// Pushing the manifest with an imageReference (the image "tag").
	if err := repo.PushReference(ctx, manifestDescriptor, bytes.NewReader(manifestBlob), imageReference); err != nil {
		log.Fatalln(fmt.Errorf("failed to push manifest with image reference: %w", err))
	}
}
