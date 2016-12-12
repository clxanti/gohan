// Copyright (C) 2015 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otto

import (
	"bytes"

	"github.com/robertkrimen/otto"
	"golang.org/x/crypto/ssh"

	"github.com/Juniper/go-netconf/netconf"
	"github.com/cloudwan/gohan/util"
	//Import otto underscore lib
	_ "github.com/robertkrimen/otto/underscore"
	"github.com/cloudwan/gohan/sync"
	"time"
	"fmt"
)

//SetUp sets up vm to with environment
func init() {
	gohanRemoteInit := func(env *Environment) {
		vm := env.VM

		builtins := map[string]interface{}{
			"gohan_netconf_open": func(call otto.FunctionCall) otto.Value {
				if len(call.ArgumentList) != 2 {
					panic("Wrong number of arguments in gohan_netconf_open call.")
				}
				rawHost, _ := call.Argument(0).Export()
				host, ok := rawHost.(string)
				if !ok {
					return otto.NullValue()
				}
				rawUserName, _ := call.Argument(1).Export()
				userName, ok := rawUserName.(string)
				if !ok {
					return otto.NullValue()
				}
				config := util.GetConfig()
				publicKeyFile := config.GetString("ssh/key_file", "")
				if publicKeyFile == "" {
					return otto.NullValue()
				}
				sshConfig := &ssh.ClientConfig{
					User: userName,
					Auth: []ssh.AuthMethod{
						util.PublicKeyFile(publicKeyFile),
					},
				}
				s, err := netconf.DialSSH(host, sshConfig)

				if err != nil {
					ThrowOttoException(&call, "Error during gohan_netconf_open: %s", err.Error())
				}
				value, _ := vm.ToValue(s)
				return value
			},
			"gohan_netconf_close": func(call otto.FunctionCall) otto.Value {
				if len(call.ArgumentList) != 1 {
					panic("Wrong number of arguments in gohan_netconf_close call.")
				}
				rawSession, _ := call.Argument(0).Export()
				s, ok := rawSession.(*netconf.Session)
				if !ok {
					ThrowOttoException(&call, "Error during gohan_netconf_close")
				}
				s.Close()
				return otto.NullValue()
			},
			"gohan_netconf_exec": func(call otto.FunctionCall) otto.Value {
				if len(call.ArgumentList) != 2 {
					panic("Wrong number of arguments in gohan_netconf_exec call.")
				}
				rawSession, _ := call.Argument(0).Export()
				s, ok := rawSession.(*netconf.Session)
				if !ok {
					return otto.NullValue()
				}
				rawCommand, _ := call.Argument(1).Export()
				command, ok := rawCommand.(string)
				if !ok {
					return otto.NullValue()
				}
				reply, err := s.Exec(netconf.RawMethod(command))
				resp := map[string]interface{}{}
				if err != nil {
					resp["status"] = "error"
					resp["output"] = err.Error()
				} else {
					resp["status"] = "success"
					resp["output"] = reply
				}
				value, _ := vm.ToValue(resp)
				return value
			},
			"gohan_ssh_open": func(call otto.FunctionCall) otto.Value {
				if len(call.ArgumentList) != 2 {
					panic("Wrong number of arguments in gohan_ssh_open call.")
				}
				rawHost, _ := call.Argument(0).Export()
				host, ok := rawHost.(string)
				if !ok {
					return otto.NullValue()
				}
				rawUserName, _ := call.Argument(1).Export()
				userName, ok := rawUserName.(string)
				if !ok {
					return otto.NullValue()
				}
				config := util.GetConfig()
				publicKeyFile := config.GetString("ssh/key_file", "")
				if publicKeyFile == "" {
					return otto.NullValue()
				}
				sshConfig := &ssh.ClientConfig{
					User: userName,
					Auth: []ssh.AuthMethod{
						util.PublicKeyFile(publicKeyFile),
					},
				}
				conn, err := ssh.Dial("tcp", host, sshConfig)
				if err != nil {
					ThrowOttoException(&call, "Error during gohan_ssh_open %s", err)
				}
				session, err := conn.NewSession()
				if err != nil {
					ThrowOttoException(&call, "Error during gohan_ssh_open %s", err)
				}
				value, _ := vm.ToValue(session)
				return value
			},
			"gohan_ssh_close": func(call otto.FunctionCall) otto.Value {
				if len(call.ArgumentList) != 1 {
					panic("Wrong number of arguments in gohan_ssh_close call.")
				}
				rawSession, _ := call.Argument(0).Export()
				s, ok := rawSession.(*ssh.Session)
				if !ok {
					ThrowOttoException(&call, "Error during gohan_ssh_close")
				}
				s.Close()
				return otto.NullValue()
			},
			"gohan_ssh_exec": func(call otto.FunctionCall) otto.Value {
				if len(call.ArgumentList) != 2 {
					panic("Wrong number of arguments in gohan_ssh_exec call.")
				}
				rawSession, _ := call.Argument(0).Export()
				s, ok := rawSession.(*ssh.Session)
				if !ok {
					return otto.NullValue()
				}
				rawCommand, _ := call.Argument(1).Export()
				command, ok := rawCommand.(string)
				if !ok {
					return otto.NullValue()
				}
				var stdoutBuf bytes.Buffer
				s.Stdout = &stdoutBuf
				err := s.Run(command)
				resp := map[string]interface{}{}
				if err != nil {
					resp["status"] = "error"
					resp["output"] = err.Error()
				} else {
					resp["status"] = "success"
					resp["output"] = stdoutBuf.String()
				}
				value, _ := vm.ToValue(resp)
				return value
			},
			"gohan_etcd_watch": func(call otto.FunctionCall) otto.Value {
				// check the number of arguments
				if len(call.ArgumentList) != 2 {
					panic("Wrong number of arguments in gohan_etcd_watch call.")
				}

				// parse first argument: path string
				rawPath, err := call.Argument(0).Export()

				if err != nil {
					ThrowOttoException(&call, "Failed to read first argument")
				}

				switch rawPath.(type) {
				case string:
				default:
					ThrowOttoException(&call, "Invalid type of first argument: expected a string")
				}

				path := rawPath.(string)

				// parse second argument: timeoutMsec int64 (int in JS)
				rawTimeoutMilliseconds, err := call.Argument(1).Export()

				if err != nil {
					ThrowOttoException(&call, "Failed to read second argument")
				}

				switch rawTimeoutMilliseconds.(type) {
				case int64:
				default:
					ThrowOttoException(&call, "Invalid type of second argument: expected an int64")
				}

				timeoutMilliseconds := rawTimeoutMilliseconds.(int64)

				// start etcd watch in goroutine
				watchEventChannel := make(chan *sync.Event, 32) // note: the size is arbitrary here
				watchStopChannel := make(chan bool, 1)
				watchDoneChannel := make(chan bool, 1)
				watchErrorChannel := make(chan error, 1)

				go func() {
					err := env.Sync.Watch(path, watchEventChannel, watchStopChannel)

					fmt.Println("*** watch done: " + fmt.Sprintf("%s %#v", err.Error(), err))

					// if there was an error, pass it through the error channel
					if err != nil {
						watchErrorChannel <- err
					}

					// mark this watch as finished, no more events will be emitted
					// after this channel is signaled
					watchDoneChannel <- true
				}()

				// start timeout timer
				timer := time.NewTimer(time.Duration(timeoutMilliseconds) * time.Millisecond)

				var event *sync.Event

				select {
				case event = <-watchEventChannel:
					// at least one event occurred, store this event and read all others if any
					fmt.Println("*** event")

				case <- timer.C:
					// even if timeout has occurred, at the time of this call there may still be
					// some events in the event channel so try to read them all
					fmt.Println("*** timeout")

				case err := <-watchErrorChannel:
					// there was an error with the etcd watch - the watch is already stopped
					// so we can safely proceed to exit
					fmt.Println("*** error")
					ThrowOttoException(&call, "Failed to watch etcd: " + err.Error())
					return otto.NullValue()
				}

				// force watch to stop in order to avoid possible generation of more watch events
				fmt.Println("*** send stop")
				watchStopChannel <- true

				// wait for the watch to finish and drain all events that are still generated
				// note: these two operations are interlocked so that etcd watch is never
				// able to overfill (and block) the channel
				jsResponse := [] map[string]interface{}{}

				done := false

				for !done {
					if event != nil {
						jsEvent := map[string]interface{}{}

  						jsEvent["action"] = event.Action
						jsEvent["data"] = event.Data
						jsEvent["key"] = event.Key

						jsResponse = append(jsResponse, jsEvent)
					}

					fmt.Println("*** wait for event or done")

					select {
					case event = <- watchEventChannel:
						continue

					case <- watchDoneChannel:
						done = true
						continue
					}
				}

				fmt.Println("*** done")

				// drain remaining events into response
				done = false

				for !done {
					select {
					case event = <- watchEventChannel:
						continue
					default:
						done = true
						continue
					}

					jsEvent := map[string]interface{}{}

					jsEvent["action"] = event.Action
					jsEvent["data"] = event.Data
					jsEvent["key"] = event.Key

					jsResponse = append(jsResponse, jsEvent)
				}

				// return an array of etcd events to JS
				fmt.Printf("*** return %v events", len(jsResponse))
				jsValue, _ := vm.ToValue(jsResponse)
				return jsValue
			},
		}
		for name, object := range builtins {
			vm.Set(name, object)
		}
	}
	RegisterInit(gohanRemoteInit)
}
