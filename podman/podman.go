package podman

import (
	"context"
	"fmt"

	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/opencontainers/runtime-spec/specs-go"
)

const ContainerSocket = "unix:///run/user/1000/podman/podman.sock" // TODO: parameterize this (using the pal config file)

func ptr[T any](v T) *T {
	return &v
}

func CommandContext(ctx context.Context, rawImage, workingDir, name string, arg ...string) ([]byte, error) {
	conn, err := bindings.NewConnection(ctx, ContainerSocket)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	// "quay.io/libpod/alpine_nginx"
	_, err = images.Pull(conn, rawImage, &images.PullOptions{
		Policy: ptr("missing"),
	}) // TODO: setup options for other environments "OS" and "Arch"

	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	s := specgen.NewSpecGenerator(rawImage, false)
	s.Command = []string{name}
	s.Command = append(s.Command, arg...)
	s.Mounts = []specs.Mount{
		{
			Destination: workingDir, // TODO: fix this ... move to a temp working dir inside, update options, mappings etc.
			Source:      workingDir,
			Type:        "bind",
			Options:     []string{"rbind", "rw"},
		},
	}
	createResponse, err := containers.CreateWithSpec(conn, s, nil)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	defer containers.Remove(conn, createResponse.ID, &containers.RemoveOptions{Force: ptr(true)}) // TODO: reuse containers? could create speedup?

	fmt.Println("Container created.")
	if err := containers.Start(conn, createResponse.ID, nil); err != nil {
		fmt.Println(err)
		return nil, err
	}
	fmt.Println("Container started.")

	stdOutChan := make(chan string)
	defer close(stdOutChan)
	stdErrChan := make(chan string)
	defer close(stdErrChan)

	stdOut := &[]byte{}
	stdErr := &[]byte{}

	go func(buf *[]byte) {
		for str := range stdOutChan {
			*buf = append(*buf, []byte(str)...)
		}
	}(stdOut)

	go func(buf *[]byte) {
		for str := range stdErrChan {
			*buf = append(*buf, []byte(str)...)
		}
	}(stdErr)

	err = containers.Logs(conn, createResponse.ID, &containers.LogOptions{
		Stderr: ptr(true),
		Stdout: ptr(true),
		Follow: ptr(true),
	}, stdOutChan, stdErrChan)

	if err != nil {
		return nil, err
	}

	// Wait for container to finish
	exitCode, err := containers.Wait(conn, createResponse.ID, &containers.WaitOptions{})
	if err != nil {
		return nil, err
	}
	fmt.Println("Container exited with code:", exitCode)

	if exitCode == 0 {
		return *stdOut, nil
	} else {
		return *stdErr, err
	}

}
