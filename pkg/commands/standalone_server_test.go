/*
Copyright © 2022 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package commands

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStart(t *testing.T) {
	tport, uport := acquirePorts(t)
	cmd := runHostResovler(t, []string{"-a", "127.0.0.1", "-t", tport, "-u", uport})
	defer cmd.Process.Kill() //nolint:errcheck

	t.Logf("Checking for TCP port is running on %v", tport)
	tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%s", tport))
	if tcpListener != nil {
		defer tcpListener.Close()
	}
	require.Errorf(t, err, "host-resolver is not listening on TCP port %s", tport)

	t.Logf("Checking for UDP port is running on %v", uport)
	udpListener, err := net.Listen("udp", fmt.Sprintf(":%s", uport))
	if udpListener != nil {
		defer udpListener.Close()
	}
	require.Errorf(t, err, "host-resolver is not listening on UDP port %s", uport)

	output := netstat(t)
	require.Contains(t, string(output), fmt.Sprintf("%v/host-resolver", cmd.Process.Pid), "Expected the same Pid")
}

func TestQueryStaticHosts(t *testing.T) {
	tport, uport := acquirePorts(t)
	cmd := runHostResovler(t, []string{
		"-a", "127.0.0.1",
		"-t", tport,
		"-u", uport,
		"-c", "host.rd.test=111.111.111.111,host2.rd.test=222.222.222.222"})
	defer cmd.Process.Kill() //nolint:errcheck

	t.Logf("Checking for TCP port on %s", tport)
	addrs, err := dnsLookup(t, tport, "tcp", "host.rd.test")
	require.NoError(t, err, "Lookup IP failed")

	expected := []net.IP{net.IPv4(111, 111, 111, 111)}
	require.ElementsMatch(t, ipToString(addrs), ipToString(expected))

	t.Logf("Checking for UDP port on %s", uport)
	addrs, err = dnsLookup(t, uport, "udp", "host2.rd.test")
	require.NoError(t, err, "Lookup IP failed")

	expected = []net.IP{net.IPv4(222, 222, 222, 222)}
	require.ElementsMatch(t, ipToString(addrs), ipToString(expected))
}

func TestQueryUpstreamServer(t *testing.T) {
	tport, uport := acquirePorts(t)
	cmd := runHostResovler(t, []string{"-a", "127.0.0.1", "-t", tport, "-u", uport, "-s", "[8.8.8.8]"})
	defer cmd.Process.Kill() //nolint:errcheck

	t.Logf("Resolving via upstream server on [TCP] --> %s", tport)
	addrs, err := dnsLookup(t, tport, "tcp", "google.ca")
	require.NoError(t, err, "Lookup IP failed")
	require.NotEmpty(t, addrs, "Expect at least an address")

	t.Logf("Resolving via upstream server on [UDP] --> %s", uport)
	addrs, err = dnsLookup(t, uport, "udp", "google.ca")
	require.NoError(t, err, "Lookup IP failed")
	require.NotEmpty(t, addrs, "Expect at least an address")
}

func runHostResovler(t *testing.T, args []string) *exec.Cmd {
	// add run command to the tip
	args = append([]string{"run"}, args...)

	cmd := exec.Command("/app/host-resolver", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	require.NoError(t, err, "host-resolver run failed")
	// little bit of pause is needed for the process to start
	// since cmd.Run() doesn't work in this situation :{
	time.Sleep(time.Second * 1)
	return cmd
}

func ipToString(ips []net.IP) (out []string) {
	for _, ip := range ips {
		out = append(out, ip.String())
	}
	return out
}

func dnsLookup(t *testing.T, resolverPort, resolverProtocol, domain string) ([]net.IP, error) {
	resolver := net.Resolver{
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialer := net.Dialer{}
			return dialer.DialContext(ctx, resolverProtocol, fmt.Sprintf(":%s", resolverPort))
		},
	}
	t.Logf("[DNS] lookup on :%s and %s -> %s", resolverPort, resolverProtocol, domain)
	// 10s timeout should be adequate
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return resolver.LookupIP(ctx, "ip4", domain)
}

// nolint: gocritic
// named returns could yield to additional code
// since the types are being converted
func acquirePorts(t *testing.T) (string, string) {
	tport, err := randomTCPPort()
	require.NoError(t, err)
	uport, err := randomUDPPort()
	require.NoError(t, err)
	return strconv.Itoa(tport), strconv.Itoa(uport)
}

func netstat(t *testing.T) []byte {
	out, err := exec.Command("netstat", "-nlp").Output()
	require.NoError(t, err, "netstat -nlp")
	t.Logf("%s\n", out)
	return out
}
