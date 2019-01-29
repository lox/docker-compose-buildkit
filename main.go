package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/lox/docker-compose-buildkit/compose"
)

/*
From docker-compose cli

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
From https://docs.docker.com/compose/compose-file/#build

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
			composeFiles stringSliceFlag
			buildArgs    stringSliceFlag
			compress     bool
			forceRemove  bool
			noCache      bool
			pull         bool
			memory       string
			parallel     bool
		)

		for _, f := range args.composeFiles {
			composeFiles = append(composeFiles, f)
		}

		// parse build command arguments
		fs := flag.NewFlagSet("build", flag.ContinueOnError)
		fs.Var(&buildArgs, "build-arg", "Set build-time variables for services")
		fs.Var(&composeFiles, "file", "The path to the docker-compose.yml file")
		fs.BoolVar(&compress, "compress", false, "Compress the build context using gzip")
		fs.BoolVar(&forceRemove, "force-remove", false, "Always remove intermediate containers.")
		fs.BoolVar(&noCache, "no-cache", false, "Do not use cache when building the image.")
		fs.BoolVar(&pull, "pull", false, "Always attempt to pull a newer version of the image.")
		fs.StringVar(&memory, "memory", "", "Sets memory limit for the build container.")
		fs.BoolVar(&parallel, "parallel", false, "Build images in parallel.")

		if err := fs.Parse(args.subcommandArgs[1:]); err != nil {
			exitWithErr(err)
		}

		// Allow COMPOSE_FILE to trump all other config
		if f := os.Getenv(`COMPOSE_FILE`); f != "" {
			composeFiles = stringSliceFlag{f}
		}

		var dockerComposeFile string

		if len(composeFiles) > 1 {
			exitWithErr(errors.New("More than one docker-compose.yml file not supported"))
		} else if len(composeFiles) == 1 {
			dockerComposeFile = composeFiles[0]
		} else {
			dockerComposeFile = `docker-compose.yml`
		}

		service := args.subcommandArgs[0]
		fmt.Printf("Building %s from %s\n", service, dockerComposeFile)

		absPath, err := filepath.Abs(dockerComposeFile)
		if err != nil {
			exitWithErr(err)
		}

		err = os.Chdir(filepath.Dir(absPath))
		if err != nil {
			exitWithErr(err)
		}

		conf, err := compose.ParseFile(absPath)
		if err != nil {
			exitWithErr(err)
		}

		serviceConf, ok := conf.Services[service]
		if !ok {
			exitWithErr(fmt.Errorf("No service named %q", service))
		}

		dockerArgs := []string{"build"}

		// Handle docker-compose directives parsed from config
		// ---------------------------------------------------

		if serviceConf.Build.Dockerfile != "" {
			dockerArgs = append(dockerArgs, `--file`, serviceConf.Build.Dockerfile)
		}

		for _, label := range serviceConf.Build.Labels {
			dockerArgs = append(dockerArgs, `--label`, label)
		}

		for _, buildArg := range serviceConf.Build.Args {
			dockerArgs = append(dockerArgs, `--build-arg`, buildArg)
		}

		dockerArgs = append(dockerArgs, serviceConf.Build.Context)

		// Handle command line arguments for build
		// ---------------------------------------------------

		if compress {
			dockerArgs = append(dockerArgs, `--compress`)
		}

		if noCache {
			dockerArgs = append(dockerArgs, `--no-cache`)
		}

		if forceRemove {
			dockerArgs = append(dockerArgs, `--force-rm`)
		}

		if pull {
			dockerArgs = append(dockerArgs, `--pull`)
		}

		// Calculate what we tag the image as
		imageTag := fmt.Sprintf("%s_%s:latest", projectName(filepath.Dir(absPath)), service)

		dockerArgs = append(dockerArgs, `--tag`, imageTag)

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
	projectName    string
	composeFiles   []string
	globalArgs     []string
	subcommand     string
	subcommandArgs []string
}

func projectName(dir string) string {
	if p := os.Getenv(`COMPOSE_PROJECT_NAME`); p != "" {
		return p
	}
	return filepath.Base(dir)
}

func parseArguments(args []string) dockerComposeArgs {
	var idx int = 1
	var result dockerComposeArgs

	var getArgWithValue = func(long, short string) (string, bool) {
		if args[idx] == `--`+long || args[idx] == `-`+short {
			val := args[idx+1]
			idx += 2
			return val, true
		} else if strings.HasPrefix(args[idx], `--`+long+`=`) || strings.HasPrefix(args[idx], `-`+short+`=`) {
			parts := strings.SplitN(args[idx], `=`, 2)
			idx++
			return parts[1], true
		}
		return "", false
	}

	for strings.HasPrefix(args[idx], "-") {
		if file, ok := getArgWithValue(`file`, `-f`); ok {
			result.composeFiles = append(result.composeFiles, file)
			continue
		}

		if name, ok := getArgWithValue(`project-name`, `p`); ok {
			result.projectName = name
			continue
		}

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
		fmt.Printf("Error: %v\n", err)
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
	*s = append(*s, value)
	return nil
}
