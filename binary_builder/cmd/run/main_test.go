/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestBuildLaunchPlanTLS(t *testing.T) {
	g := NewWithT(t)

	raw := []byte(`{
		"chaincode_id": "cc:abc",
		"peer_address": "peer0:7052",
		"client_cert": "CLIENTCERT",
		"client_key": "CLIENTKEY",
		"root_cert": "ROOTCERT",
		"mspid": "Org1MSP"
	}`)

	plan, err := buildLaunchPlan("/bld/chaincode", "/launch", raw, nil)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(plan.Path).To(Equal("/bld/chaincode"))
	g.Expect(plan.Args).To(Equal([]string{"/bld/chaincode", "-peer.address=peer0:7052"}))

	g.Expect(plan.Env).To(ContainElement("METADATA=" + string(raw)))
	g.Expect(plan.Env).To(ContainElement("CORE_CHAINCODE_ID_NAME=cc:abc"))
	g.Expect(plan.Env).To(ContainElement("CORE_PEER_LOCALMSPID=Org1MSP"))
	g.Expect(plan.Env).To(ContainElement("CORE_PEER_TLS_ENABLED=true"))
	g.Expect(plan.Env).To(ContainElement("CORE_TLS_CLIENT_CERT_FILE=/launch/client.crt"))
	g.Expect(plan.Env).To(ContainElement("CORE_TLS_CLIENT_KEY_FILE=/launch/client.key"))
	g.Expect(plan.Env).To(ContainElement("CORE_PEER_TLS_ROOTCERT_FILE=/launch/root.crt"))

	g.Expect(plan.Files).To(HaveKeyWithValue("/launch/client.crt", []byte("CLIENTCERT")))
	g.Expect(plan.Files).To(HaveKeyWithValue("/launch/client.key", []byte("CLIENTKEY")))
	g.Expect(plan.Files).To(HaveKeyWithValue("/launch/root.crt", []byte("ROOTCERT")))
}

func TestBuildLaunchPlanNoTLS(t *testing.T) {
	g := NewWithT(t)

	raw := []byte(`{"chaincode_id":"cc:abc","peer_address":"peer0:7052","mspid":"Org1MSP"}`)

	plan, err := buildLaunchPlan("/bld/chaincode", "/launch", raw, nil)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(plan.Env).To(ContainElement("CORE_PEER_TLS_ENABLED=false"))
	g.Expect(plan.Env).NotTo(ContainElement(ContainSubstring("CORE_TLS_CLIENT_CERT_FILE")))
	g.Expect(plan.Files).To(BeEmpty())
}

func TestBuildLaunchPlanPreservesBaseEnv(t *testing.T) {
	g := NewWithT(t)

	raw := []byte(`{"chaincode_id":"cc","peer_address":"a:1"}`)
	plan, err := buildLaunchPlan("/bld/chaincode", "/launch", raw, []string{"PATH=/usr/bin"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(plan.Env).To(ContainElement("PATH=/usr/bin"))
}

func TestBuildLaunchPlanInvalidJSON(t *testing.T) {
	g := NewWithT(t)

	_, err := buildLaunchPlan("/bld/chaincode", "/launch", []byte("not json"), nil)
	g.Expect(err).To(HaveOccurred())
}

func TestRunTooFewArgs(t *testing.T) {
	g := NewWithT(t)
	g.Expect(run([]string{"run", "onlyone"}, nil)).To(HaveOccurred())
}
