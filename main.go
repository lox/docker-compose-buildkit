package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/lox/docker-compose-buildkit/compose"
)

/*
Usage: build [options] [--build-arg key=val...] [SERVICE...]

Options:
    --compress              Compress the build context using gzip.
    --force-rm              Always remove intermediate containers.
    --no-cache              Do not use cache when building the image.
    --pull                  Always attempt to pull a newer version of the image.
    -m, --memory MEM        Sets memory limit for the build container.
    --build-arg key=val     Set build-time variables for services.
	--parallel              Build images in parallel.
*/

/*
build
  context
  dockerfile
  args
  cache_from
  labels
  shm_size
  target
*/

func main() {
	args := parseArguments(os.Args)

	// detect a build command and use docker binary instead
	if args.subcommand == `build` {
		var (
			buildArgs   stringSliceFlag
			compress    bool
			forceRemove bool
			pull        bool
			memory      string
			parallel    bool
		)

		fs := flag.NewFlagSet("build", flag.ContinueOnError)
		fs.Var(&buildArgs, "build-arg", "Set build-time variables for services")
		fs.BoolVar(&compress, "compress", false, "Compress the build context using gzip")
		fs.BoolVar(&forceRemove, "force-remove", false, "Always remove intermediate containers.")
		fs.BoolVar(&pull, "pull", false, "Always attempt to pull a newer version of the image.")
		fs.StringVar(&memory, "memory", "", "Sets memory limit for the build container.")
		fs.BoolVar(&parallel, "parallel", false, "Build images in parallel.")

		if err := fs.Parse(args.subcommandArgs[1:]); err != nil {
			exitWithErr(err)
		}

		fmt.Printf("Args: %#v\n", buildArgs)

		service := args.subcommandArgs[0]
		fmt.Printf("Building %s\n", service)

		conf, err := compose.ParseFile("docker-compose.yml")
		if err != nil {
			exitWithErr(err)
		}

		serviceConf, ok := conf.Services[service]
		if !ok {
			exitWithErr(fmt.Errorf("No service named %q", service))
		}

		dockerArgs := []string{"build"}

		if serviceConf.Build.Dockerfile != "" {
			dockerArgs = append(dockerArgs, `--file`, serviceConf.Build.Dockerfile)
		}

		// handle labels from config
		for _, label := range serviceConf.Build.Labels {
			dockerArgs = append(dockerArgs, `--label`, label)
		}

		// handle build-args from config
		for _, buildArg := range serviceConf.Build.Args {
			dockerArgs = append(dockerArgs, `--build-arg`, buildArg)
		}

		if compress {
			dockerArgs = append(dockerArgs, `--compress`)
		}

		if forceRemove {
			dockerArgs = append(dockerArgs, `--force-rm`)
		}

		if pull {
			dockerArgs = append(dockerArgs, `--pull`)
		}

		dockerArgs = append(dockerArgs, serviceConf.Build.Context)

		fmt.Printf("Conf: %#v\n", conf)
		fmt.Printf("Args: %#v\n", dockerArgs)

		cmd := exec.Command(`docker`, dockerArgs...)
		cmd.Env = append(os.Environ(), `DOCKER_BUILDKIT=1`)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			exitWithErr(err)
		}

		os.Exit(0)
	}

	cmd := exec.Command(`docker-compose`, os.Args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		exitWithErr(err)
	}
}

type dockerComposeArgs struct {
	globalArgs     []string
	subcommand     string
	subcommandArgs []string
}

func parseArguments(args []string) dockerComposeArgs {
	var idx int = 1
	var result dockerComposeArgs

	for strings.HasPrefix(args[idx], "-") {
		result.globalArgs = append(result.globalArgs, args[idx])
		idx++
	}

	if idx >= len(args) {
		return result
	}

	result.subcommand = args[idx]
	idx++

	for idx < len(args) {
		result.subcommandArgs = append(result.subcommandArgs, args[idx])
		idx++
	}

	return result
}

func exitWithErr(err error) {
	var waitStatus syscall.WaitStatus
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		os.Exit(1)
	}
	waitStatus = exitError.Sys().(syscall.WaitStatus)
	os.Exit(waitStatus.ExitStatus())
}

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *stringSliceFlag) Set(value string) error {
	fmt.Printf("Set %s\n", value)
	*s = append(*s, value)
	return nil
}
