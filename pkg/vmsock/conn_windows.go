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
	"fmt"
	"net"

	"github.com/Microsoft/go-winio"
	"github.com/linuxkit/virtsock/pkg/hvsock"
)

func Listen() (net.Listener, error) {
	vmGuid, err := vmGuid()
	if err != nil {
		return nil, fmt.Errorf("Listen, could not use VM GUID: %v", err)
	}
	svcPort, err := hvsock.GUIDFromString(winio.VsockServiceID(HostListenPort).String())
	if err != nil {
		return nil, fmt.Errorf("Listen, could not parse Hyper-v service GUID: %v", err)
	}

	addr := hvsock.Addr{
		VMID:      vmGuid,
		ServiceID: svcPort,
	}

	return hvsock.Listen(addr)
}
