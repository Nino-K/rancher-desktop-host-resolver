# Host Resolver
A stub DNS resolver that runs on the host machine on Linux, macOS, and Windows. The main goal behind this stub resolver is more robuset handling of domain name resolutions when VPN split tunnel is setup.  

![b822615f-af15-4997-981b-53a6f1153d81 sketchpad (1)](https://user-images.githubusercontent.com/10409174/161309152-e048d73f-fbe6-42a2-a409-c29a7de7f03a.jpeg)


## Run

```bash
/host-resolver run -a 127.0.0.1 -t 54 -u 53 -c "host.rd.internal=111.111.111.111,host2.rd.internal=222.222.222.222"
```
NOTE: If ports are not provided, host resolver will listen on random ports.

## Test

You can run the tests:

### Run Locally 

TODO: add a way to run locally since we will need this for the CI

```bash
go test -v ./...
```
Note: this may require sudo

### Run In Container

Or run them in a container. 
```bash
docker build -t host-resolver:latest . && docker run --dns 127.0.0.1 -it host-resolver:latest
```
Note: Run with `--dns` flag, this overrides the `/etc/resolv.conf` in the running container. 
