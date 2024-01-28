package core

import (
	"archive/tar"
	"bytes"
	"os"

	docker "github.com/fsouza/go-dockerclient"
)

func BuildTestImage(client *docker.Client, name string) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	tw.WriteHeader(&tar.Header{Name: "Dockerfile"})
	tw.Write([]byte("FROM alpine\n"))
	tw.Close()

	return client.BuildImage(docker.BuildImageOptions{
		Name:         name,
		Remote:       "github.com/mcuadros/ofelia",
		InputStream:  &buf,
		OutputStream: os.Stdout,
	})
}
