// Parts of this code are inspired by the ORAS Project documentation.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

func pull(registryName string, repositoryName string, imageReference string, output string, username string, password string) {
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

	log.Printf("Pulling blobs from %s/%s:%s\n", registryName, repositoryName, imageReference)

	// Obtains the manifest descriptor for the specified imageReference.
	manifestDescriptor, rc, err := repo.FetchReference(ctx, imageReference)
	if err != nil {
		panic(err)
	}
	defer rc.Close()

	// Read the bytes of the manifest descriptor from the io.ReadCloser.
	pulledContent, err := content.ReadAll(rc, manifestDescriptor)
	if err != nil {
		panic(err)
	}

	// JSON Decodes the bytes read into an OCI Manifest.
	var pulledManifest ocispec.Manifest
	if err := json.Unmarshal(pulledContent, &pulledManifest); err != nil {
		log.Fatalln(fmt.Errorf("failed to json decode the pulled oci manifest: %w", err))
	}

	// Check if the Output Path exists before writing to it.
	if _, err := os.Stat(output); os.IsNotExist(err) {
		log.Fatalln(fmt.Errorf("specified output path '%s' does not exits: %w", output, err))
	}

	// Loop over the layers found in the OCI Manifest to download their content (blob).
	for _, layer := range pulledManifest.Layers {
		filename := layer.Annotations["org.opencontainers.image.title"]

		log.Printf("Downloading blob %s [%s]\n", filename, byteCountIEC(layer.Size))

		pulledBlob, err := content.FetchAll(ctx, repo, layer)
		if err != nil {
			log.Fatalln(fmt.Errorf("failed to fetch layer content (blob): %w", err))
		}

		if err := os.WriteFile(path.Join(output, filename), pulledBlob, 0644); err != nil {
			log.Fatalln(fmt.Errorf("failed to write layer content (blob) to file %s: %w", path.Join(output, filename), err))
		}
	}
}
