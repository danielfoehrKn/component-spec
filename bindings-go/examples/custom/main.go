// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
)

func main() {
	data := []byte(`
meta:
  schemaVersion: 'v2'

component:
  name: 'github.com/gardener/gardener'
  version: 'v1.7.2'
  provider: internal
  repositoryContexts: []
  sources: []
  componentReferences: []

  resources:
  - name: 'ftpRes'
    version: 'v1.7.2'
    type: 'custom1'
    relation: local
    access:
      type: 'x-ftp'
      url: ftp://example.com/my-resource

  - name: 'nodeMod'
    version: '0.0.1'
    type: 'nodeModule'
    relation: external
    access:
      type: 'npm'
      nodeModule: my-module
      version: 0.0.1
`)

	component := &v2.ComponentDescriptor{}
	err := codec.Decode(data, component)
	check(err)

	res, err := component.GetLocalResource("custom1", "ftpRes", "v1.7.2")
	check(err)
	// by default all types are serialized as unstructured type
	ftpAccess := res.Access
	fmt.Println(ftpAccess.Object["url"]) // prints: ftp://example.com/my-resource

	// By default unknown types are decoded with the JSONDecoder.
	// A specific decoder can be registered by adding it to known types.
	knownTypes := v2.KnownAccessTypes
	knownTypes.Register(NPMType, npmCodec)
	accessTypeCodec := v2.NewCodec(knownTypes, nil, nil)
	res, err = component.GetExternalResource("nodeModule", "nodeMod", "0.0.1")
	check(err)
	npmAccess := &NPMAccess{}
	check(accessTypeCodec.Decode(res.Access.Raw, npmAccess))
	fmt.Println(npmAccess.NodeModule) // prints: my-module
}

func check(err error) {
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

const NPMType = "npm"

type NPMAccess struct {
	v2.ObjectType
	NodeModule string `json:"nodeModule"`
	Version    string `json:"version"`
}

var _ v2.TypedObjectAccessor = &NPMAccess{}

func (n NPMAccess) GetData() ([]byte, error) {
	return json.Marshal(n)
}

func (n *NPMAccess) SetData(bytes []byte) error {
	var newNPM NPMAccess
	if err := json.Unmarshal(bytes, &newNPM); err != nil {
		return err
	}

	n.NodeModule = newNPM.NodeModule
	n.Version = newNPM.Version
	return nil
}

var npmCodec = &v2.TypedObjectCodecWrapper{
	TypedObjectDecoder: v2.TypedObjectDecoderFunc(func(data []byte, into v2.TypedObjectAccessor) error {
		obj, ok := into.(*NPMAccess)
		if !ok {
			return errors.New("undecodable type")
		}
		var npm NPMAccess
		if err := json.Unmarshal(data, &npm); err != nil {
			return err
		}
		*obj = npm
		return nil
	}),
	TypedObjectEncoder: v2.TypedObjectEncoderFunc(func(accessor v2.TypedObjectAccessor) ([]byte, error) {
		npm, ok := accessor.(*NPMAccess)
		if !ok {
			return nil, fmt.Errorf("accessor is not of type %s", NPMType)
		}
		return json.Marshal(npm)
	}),
}
