package nodeexec

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
)

//
// docker container prune && docker volume prune
// ./swarm-node/scripts/ecr-build.sh
// docker run -v /var/run/docker.sock:/var/run/docker.sock -v pipelines:/pipelines -e CONSOLE_LOGGING=1 -it axond
//

type InputStage struct {
	Platform      string
	Files         []File
	EntryFilename string
	EntryArgs     []string
}

type File struct {
	Filename string
	Size     int64
	Contents io.Reader
}

type ContainerStage struct {
	ImageName  string
	Dockerfile string
	Files      []File
}

var platforms = map[string]struct {
	BaseImage  string
	RunCommand string
}{
	"python": {
		"python:alpine",
		"python",
	},
	"node": {
		"node:12-alpine",
		"node",
	},
}

func StartPipeline(inputStages []InputStage) (io.WriteCloser, io.ReadCloser, error) {
	// Create the Docker client
	c, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	// Prepare the host machine
	hostEnv, err := prepareHostEnv(len(inputStages))
	if err != nil {
		return nil, nil, err
	}

	// Create containers for each pipeline stage
	containerStages, err := inputStagesToContainerStages(inputStages, hostEnv)
	if err != nil {
		return nil, nil, err
	}

	var containerIDs []string
	for _, stage := range containerStages {
		err = buildImage(c, stage)
		if err != nil {
			return nil, nil, err
		}

		containerID, err := createContainer(c, stage, hostEnv)
		if err != nil {
			return nil, nil, err
		}
		containerIDs = append(containerIDs, containerID)
	}

	// Start the containers
	for _, containerID := range containerIDs {
		err = startContainer(c, containerID)
		if err != nil {
			return nil, nil, err
		}
	}

	// Attempting to open a named pipe in O_RDONLY mode prior to something starting to write to it is a
	// blocking operation.  As a result, we have to open the output pipe after containers have been started.
	err = hostEnv.prepareOutput()
	if err != nil {
		return nil, nil, err
	}

	// @@TODO: destroy containers
	return hostEnv.In, hostEnv.Out, nil
}

type HostEnv struct {
	Root    string
	In      io.WriteCloser
	Out     io.ReadCloser
	outPath string
}

// Attempting to open a named pipe in O_RDONLY mode prior to something starting to write to it is a
// blocking operation.  As a result, we have to open the output pipe after containers have been started.
func (e *HostEnv) prepareOutput() error {
	pipelineOut, err := os.OpenFile(e.outPath, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		return errors.WithStack(err)
	}
	e.Out = pipelineOut
	log.Infoln("[exec]   - PIPELINE_OUT opened")
	return nil
}

func prepareHostEnv(numStages int) (*HostEnv, error) {
	log.Infoln("[exec] preparing host env...")

	// Create sandbox dir to isolate this from other running pipelines
	rootDir, err := ioutil.TempDir("/pipelines", "xyzzy-")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Infoln("[exec]   - rootDir:", rootDir)

	// Create pipeline input and output pipes
	var (
		pipelineInPath  = filepath.Join(rootDir, "PIPELINE_IN")
		pipelineOutPath = filepath.Join(rootDir, "PIPELINE_OUT")
	)

	err = syscall.Mkfifo(pipelineInPath, uint32(os.ModeNamedPipe|0644))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	log.Infoln("[exec]   - PIPELINE_IN created")

	err = syscall.Mkfifo(pipelineOutPath, uint32(os.ModeNamedPipe|0644))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	log.Infoln("[exec]   - PIPELINE_OUT created")

	for i := 0; i < numStages-1; i++ {
		pipePath := filepath.Join(rootDir, fmt.Sprintf("pipe%v", i))
		err = syscall.Mkfifo(pipePath, uint32(os.ModeNamedPipe|0644))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		log.Infof("[exec]   - pipe%v created", i)
	}

	pipelineIn, err := os.OpenFile(pipelineInPath, os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	log.Infoln("[exec]   - PIPELINE_IN opened")

	log.Infoln("[exec] done preparing host env.")

	return &HostEnv{
		Root: rootDir,
		In:   pipelineIn,
		// Out:  pipelineOut,
		outPath: pipelineOutPath,
	}, nil
}

func inputStagesToContainerStages(inputStages []InputStage, hostEnv *HostEnv) ([]ContainerStage, error) {
	var containerStages []ContainerStage

	for i, inputStage := range inputStages {
		platform := platforms[inputStage.Platform]
		dockerfile := []string{
			`FROM ` + platform.BaseImage,
			`RUN mkdir /data`,
			`VOLUME /data/shared`,
			`RUN mkdir -p /data/shared`,
		}

		if i == 0 {
			dockerfile = append(dockerfile, fmt.Sprintf(`RUN ln -s /data/shared/%v/PIPELINE_IN /data/in`, filepath.Base(hostEnv.Root)))
			dockerfile = append(dockerfile, fmt.Sprintf(`RUN ln -s /data/shared/%v/pipe%v /data/out`, filepath.Base(hostEnv.Root), i))

		} else if i > 0 && i < len(inputStages)-1 {
			dockerfile = append(dockerfile, fmt.Sprintf(`RUN ln -s /data/shared/%v/pipe%v /data/in`, filepath.Base(hostEnv.Root), i-1))
			dockerfile = append(dockerfile, fmt.Sprintf(`RUN ln -s /data/shared/%v/pipe%v /data/out`, filepath.Base(hostEnv.Root), i))

		} else {
			dockerfile = append(dockerfile, fmt.Sprintf(`RUN ln -s /data/shared/%v/pipe%v /data/in`, filepath.Base(hostEnv.Root), i-1))
			dockerfile = append(dockerfile, fmt.Sprintf(`RUN ln -s /data/shared/%v/PIPELINE_OUT /data/out`, filepath.Base(hostEnv.Root)))
		}

		for _, file := range inputStage.Files {
			// @@TODO: put code files in /code?
			// @@TODO: preserve dir structure
			dockerfile = append(dockerfile, `ADD `+file.Filename+` /`)
		}
		cmd := append([]string{platform.RunCommand, inputStage.EntryFilename}, inputStage.EntryArgs...)
		cmdStr, err := json.Marshal(cmd)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		dockerfile = append(dockerfile, `CMD `+string(cmdStr))

		containerStages = append(containerStages, ContainerStage{
			ImageName:  fmt.Sprintf("xyzzy-%v", i),
			Dockerfile: strings.Join(dockerfile, "\n"),
			Files:      inputStage.Files,
		})
	}
	return containerStages, nil
}

func buildImage(c *client.Client, stage ContainerStage) error {
	log.Infoln("[exec] building image", stage.ImageName, "...")
	log.Infoln("[exec] Dockerfile:\n", stage.Dockerfile)

	// Fallback in case files aren't closed normally
	// defer func() {
	// 	for _, file := range stage.Files {
	// 		file.Contents.Close()
	// 	}
	// }()

	var (
		dockerfile = []byte(stage.Dockerfile)
		tarBuffer  = bytes.NewBuffer(nil)
		tw         = tar.NewWriter(tarBuffer)
	)

	tw.WriteHeader(&tar.Header{
		Name: "Dockerfile",
		Size: int64(len(dockerfile)),
	})
	_, err := tw.Write(dockerfile)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, file := range stage.Files {
		err = tw.WriteHeader(&tar.Header{
			Name: file.Filename,
			Size: file.Size,
		})
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = io.Copy(tw, file.Contents)
		if err != nil {
			return errors.WithStack(err)
		}

		// file.Contents.Close()
	}

	err = tw.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	resp, err := c.ImageBuild(context.Background(), tarBuffer, types.ImageBuildOptions{Remove: true, ForceRemove: true, NoCache: true, Tags: []string{stage.ImageName}})
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		log.Infoln("[exec] docker:", scanner.Text())
	}

	log.Infoln("[exec] done building image", stage.ImageName)

	return nil
}

func createContainer(c *client.Client, stage ContainerStage, hostEnv *HostEnv) (string, error) {
	log.Infoln("[exec] creating container", stage.ImageName, "...")

	containerConfig := &container.Config{
		// Hostname        string              // Hostname
		// Domainname      string              // Domainname
		// User            string              // User that will run the command(s) inside the container, also support user:group
		// AttachStdin     bool                // Attach the standard input, makes possible user interaction
		// AttachStdout    bool                // Attach the standard output
		// AttachStderr    bool                // Attach the standard error
		// ExposedPorts    nat.PortSet         `json:",omitempty"` // List of exposed ports
		// Tty             bool                // Attach standard streams to a tty, including stdin if it is not closed.
		// OpenStdin       bool                // Open stdin
		// StdinOnce       bool                // If true, close stdin after the 1 attached client disconnects.
		// HostEnv             []string            // List of environment variable to set in the container
		// Cmd             strslice.StrSlice   // Command to run when starting the container
		// Healthcheck     *HealthConfig       `json:",omitempty"` // Healthcheck describes how to check the container is healthy
		// ArgsEscaped     bool                `json:",omitempty"` // True if command is already escaped (meaning treat as a command line) (Windows specific).
		Image: stage.ImageName, // Name of the image as it was passed by the operator (e.g. could be symbolic)
		// Volumes         map[string]struct{} // List of volumes (mounts) used for the container
		Volumes: map[string]struct{}{
			// vol.Name: {},
			"pipelines": {},
		},
		// WorkingDir      string              // Current directory (PWD) in the command will be launched
		// Entrypoint      strslice.StrSlice   // Entrypoint to run when starting the container
		// NetworkDisabled bool                `json:",omitempty"` // Is network disabled
		// MacAddress      string              `json:",omitempty"` // Mac Address of the container
		// OnBuild         []string            // ONBUILD metadata that were defined on the image Dockerfile
		// Labels          map[string]string   // List of labels set to this container
		// StopSignal      string              `json:",omitempty"` // Signal to stop a container
		// StopTimeout     *int                `json:",omitempty"` // Timeout (in seconds) to stop a container
		// Shell           strslice.StrSlice   `json:",omitempty"` // Shell for shell-form of RUN, CMD, ENTRYPOINT
	}

	hostConfig := &container.HostConfig{
		// Applicable to all platforms
		// Binds           []string      // List of volume bindings for this container
		// ContainerIDFile string        // File (path) where the containerId is written
		// LogConfig       LogConfig     // Configuration of the logs for this container
		// NetworkMode     NetworkMode   // Network mode to use for the container
		// PortBindings    nat.PortMap   // Port mapping between the exposed port (container) and the host
		// RestartPolicy   RestartPolicy // Restart policy to be used for the container
		// AutoRemove      bool          // Automatically remove container when it exits
		// VolumeDriver    string        // Name of the volume driver used to mount volumes
		// VolumesFrom     []string      // List of volumes to take from other container
		// VolumesFrom: []string{"axon-node"},

		// // Applicable to UNIX platforms
		// CapAdd          strslice.StrSlice // List of kernel capabilities to add to the container
		// CapDrop         strslice.StrSlice // List of kernel capabilities to remove from the container
		// Capabilities    []string          `json:"Capabilities"` // List of kernel capabilities to be available for container (this overrides the default set)
		// CgroupnsMode    CgroupnsMode      // Cgroup namespace mode to use for the container
		// DNS             []string          `json:"Dns"`        // List of DNS server to lookup
		// DNSOptions      []string          `json:"DnsOptions"` // List of DNSOption to look for
		// DNSSearch       []string          `json:"DnsSearch"`  // List of DNSSearch to look for
		// ExtraHosts      []string          // List of extra hosts
		// GroupAdd        []string          // List of additional groups that the container process will run as
		// IpcMode         IpcMode           // IPC namespace to use for the container
		// Cgroup          CgroupSpec        // Cgroup to use for the container
		// Links           []string          // List of links (in the name:alias form)
		// OomScoreAdj     int               // Container preference for OOM-killing
		// PidMode         PidMode           // PID namespace to use for the container
		// Privileged      bool              // Is the container in privileged mode
		// PublishAllPorts bool              // Should docker publish all exposed port for the container
		// ReadonlyRootfs  bool              // Is the container root filesystem in read-only
		// SecurityOpt     []string          // List of string values to customize labels for MLS systems, such as SELinux.
		// StorageOpt      map[string]string `json:",omitempty"` // Storage driver options per container.
		// Tmpfs           map[string]string `json:",omitempty"` // List of tmpfs (mounts) used for the container
		// UTSMode         UTSMode           // UTS namespace to use for the container
		// UsernsMode      UsernsMode        // The user namespace to use for the container
		// ShmSize         int64             // Total shm memory usage
		// Sysctls         map[string]string `json:",omitempty"` // List of Namespaced sysctls used for the container
		// Runtime         string            `json:",omitempty"` // Runtime to use with this container

		// // Applicable to Windows
		// ConsoleSize [2]uint   // Initial console size (height,width)
		// Isolation   Isolation // Isolation technology of the container (e.g. default, hyperv)

		// // Contains container's resources (cgroups, ulimits)
		// Resources

		// // Mounts specs used by the container
		Mounts: []mount.Mount{
			// {
			// 	Type: mount.TypeBind,
			// 	// Source specifies the name of the mount. Depending on mount type, this
			// 	// may be a volume name or a host path, or even ignored.
			// 	// Source is not supported for tmpfs (must be an empty value)
			// 	Source:      hostEnv.Root,
			// 	Target:      "/data/shared",
			// 	ReadOnly:    false,
			// 	Consistency: mount.ConsistencyFull,

			// 	BindOptions: &mount.BindOptions{
			// 		Propagation: mount.PropagationPrivate,
			// 	},
			// 	// VolumeOptions *VolumeOptions `json:",omitempty"`
			// 	// TmpfsOptions  *TmpfsOptions  `json:",omitempty"`
			// },
			{
				Type: mount.TypeVolume,
				// Source specifies the name of the mount. Depending on mount type, this
				// may be a volume name or a host path, or even ignored.
				// Source is not supported for tmpfs (must be an empty value)
				Source:      "pipelines", //vol.Name,
				Target:      "/data/shared",
				ReadOnly:    false,
				Consistency: mount.ConsistencyFull,

				VolumeOptions: &mount.VolumeOptions{
					NoCopy: true,
				},
				// TmpfsOptions  *TmpfsOptions  `json:",omitempty"`
			},
		},

		// // MaskedPaths is the list of paths to be masked inside the container (this overrides the default set of paths)
		// MaskedPaths []string

		// // ReadonlyPaths is the list of paths to be set as read-only inside the container (this overrides the default set of paths)
		// ReadonlyPaths []string

		// // Run a custom init inside the container, if null, use the daemon's configured settings
		// Init *bool `json:",omitempty"`
	}

	networkingConfig := &network.NetworkingConfig{}

	containerName := ""

	resp, err := c.ContainerCreate(context.Background(), containerConfig, hostConfig, networkingConfig, containerName)
	if err != nil {
		return "", errors.WithStack(err)
	}

	containerID := resp.ID

	log.Infoln("[exec] done creating container", stage.ImageName)

	return containerID, nil
}

func startContainer(c *client.Client, containerID string) error {
	err := c.ContainerStart(context.Background(), containerID, types.ContainerStartOptions{})
	return errors.WithStack(err)
}
