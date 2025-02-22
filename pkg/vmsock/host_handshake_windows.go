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

package vmsock

import (
	"errors"
	"fmt"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/linuxkit/virtsock/pkg/hvsock"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/registry"
)

const timeoutSeconds = 10

// vmGuid retrieves the GUID for a correct hyper-v VM (most likely WSL).
// It performs a handshake with a running vsock-peer in the WSL distro
// to make sure we establish the AF_VSOCK connection with a right VM.
func vmGuid() (hvsock.GUID, error) {
	key, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion\HostComputeService\VolatileStore\ComputeSystem`,
		registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return hvsock.GUIDZero, fmt.Errorf("could not open registry key, is WSL VM running? %v", err)
	}
	defer key.Close()

	names, err := key.ReadSubKeyNames(0)
	if err != nil {
		return hvsock.GUIDZero, fmt.Errorf("machine IDs can not be read in registry: %v", err)
	}
	if len(names) == 0 {
		return hvsock.GUIDZero, errors.New("no running WSL VM found")
	}

	found := make(chan hvsock.GUID, len(names))
	done := make(chan bool, len(names))
	defer close(done)

	for _, name := range names {
		vmGuid, err := hvsock.GUIDFromString(name)
		if err != nil {
			log.Errorf("invalid VM name: [%s], err: %v", name, err)
			continue
		}
		go handshake(vmGuid, found, done)
	}
	return tryFindGuid(found)
}

// tryFindGuid waits on a found chanel to receive a GUID until
// deadline of 10s is reached
func tryFindGuid(found chan hvsock.GUID) (hvsock.GUID, error) {
	bailOut := time.After(time.Second * timeoutSeconds)
	for {
		select {
		case vmGuid := <-found:
			return vmGuid, nil
		case <-bailOut:
			return hvsock.GUIDZero, errors.New("could not find vsock-peer process on any hyper-v VM(s)")
		}
	}
}

// handshake attempts to perform a handshake by verifying the seed with a running
// af_vsock peer in WSL distro, it attempts once per second
func handshake(vmGuid hvsock.GUID, found chan hvsock.GUID, done chan bool) {
	svcPort, err := hvsock.GUIDFromString(winio.VsockServiceID(PeerHandshakePort).String())
	if err != nil {
		log.Errorf("hostHandshake parsing svc port: %v", err)
	}
	addr := hvsock.Addr{
		VMID:      vmGuid,
		ServiceID: svcPort,
	}

	attempInterval := time.NewTicker(time.Second * 1)
	attempt := 1
	for {
		select {
		case <-done:
			log.Infof("attempt to handshake with [%s], goroutine is terminated", vmGuid.String())
			return
		case <-attempInterval.C:
			conn, err := hvsock.Dial(addr)
			if err != nil {
				attempt++
				continue
			}
			seed := make([]byte, len(SeedPhrase))
			_, err = conn.Read(seed)
			if err != nil {
				log.Errorf("hosthandshake attempt to read the seed: %v", err)
			}
			if err := conn.Close(); err != nil {
				log.Errorf("hosthandshake closing connection: %v", err)
			}
			if string(seed) == SeedPhrase {
				log.Infof("successfully estabilished a handshake with a peer: %s", vmGuid.String())
				found <- vmGuid
				return
			}
		}
	}
}
