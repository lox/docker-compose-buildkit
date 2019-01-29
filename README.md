# docker-compose-buildkit

This is a shim for `docker-compose` that uses the new Buildkit feature in docker when `docker-compose build` is invoked.

It's a band-aid until docker-compose adds support: https://github.com/moby/buildkit/issues/685

## What is supported?

The following options are read from the command line:

 * [x] `--file FILE             The path to the docker-compose.yml file`
 * [x] `--compress              Compress the build context using gzip.`
 * [x] `--force-rm              Always remove intermediate containers.`
 * [x] `--no-cache              Do not use cache when building the image.`
 * [x] `--pull                  Always attempt to pull a newer version of the image.`
 * [ ] `-m, --memory MEM        Sets memory limit for the build container.`
 * [x] `--build-arg key=val     Set build-time variables for services.`
 * [ ] `--parallel              Build images in parallel.`


Also the docker-compose.yml is parsed and the settings for the build for the relevant service are examined. The following are supported:

 * [ ] `image`
 * [ ] `build.context`
 * [ ] `build.dockerfile`
 * [ ] `build.args`
 * [ ] `build.cache_from`
 * [ ] `build.labels`
 * [ ] `build.shm_size`
 * [ ] `build.target`

