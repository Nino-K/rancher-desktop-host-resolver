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
package cmd

import (
	"context"

	"github.com/rancher-sandbox/rancher-desktop-host-resolver/pkg/vmsock"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

const defaultPort = 53

// peerCmd represents the vsock-peer process that runs in WSL Distro, it listens on
// specified IP address and ports inside a WSL distro for all incoming DNS queries.
// It forwards the queries over the AF_VSOCK connection to the receiving vsock-host
// that is running on host.
var (
	peerCmd = &cobra.Command{
		Use:   "vsock-peer",
		Short: "Host-resolver vsock-peer process",
		Long: `Peer process to handle incoming TCP/UDP DNS queries over the
AF_VSOCK connections from inside of the WSL VM. It is also a stub DNS forwarder for the host-resovler.

--------------------HOST-------------------------------------WSL DISTRO------------
| vsock-host | <----- AF_VSOCK -----> [ VM ] <----- AF_VSOCK -----> | vsock-peer |
-----------------------------------------------------------------------------------`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// We will always listen for handshake connections from the host (server) in case of restarts
			go vmsock.PeerHandshake()

			addr, err := cmd.Flags().GetString("listen-address")
			if err != nil {
				return err
			}
			tcpPort, err := cmd.Flags().GetInt("tcp-port")
			if err != nil {
				return err
			}
			udpPort, err := cmd.Flags().GetInt("udp-port")
			if err != nil {
				return err
			}
			errs, _ := errgroup.WithContext(context.Background())
			errs.Go(func() error {
				return vmsock.ListenTCP(addr, tcpPort)
			})
			errs.Go(func() error {
				return vmsock.ListenUDP(addr, udpPort)
			})

			return errs.Wait()
		},
	}
)

func init() {
	peerCmd.Flags().StringP("listen-address", "a", "127.0.0.1", "Address to listen on, \"127.0.0.1:dnsPort\" if empty.")
	peerCmd.Flags().IntP("tcp-port", "t", defaultPort, "TCP port to listen on, default is 53.")
	peerCmd.Flags().IntP("udp-port", "u", defaultPort, "UDP port to listen on, default is 53.")
	rootCmd.AddCommand(peerCmd)
}
