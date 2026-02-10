// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package provider

// TODO: Add acceptance tests for Superset provider

//import (
//	"math/rand"
//	"os"
//	"testing"
//
//	"github.com/hashicorp/terraform-plugin-framework/providerserver"
//	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
//)
//
//var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
//	"superset": providerserver.NewProtocol6WithError(New("test")()),
//}
//
//func testAccPreCheck(t *testing.T) {
//	t.Helper()
//
//	if os.Getenv("TF_ACC") != "1" {
//		t.Skip("TF_ACC is not set to 1; skipping acceptance tests")
//	}
//	if os.Getenv("SUPERSET_ENDPOINT") == "" || os.Getenv("SUPERSET_TOKEN") == "" {
//		t.Skip("SUPERSET_ENDPOINT and SUPERSET_TOKEN must be set for acceptance tests")
//	}
//}
//
//func testAccRandSuffix() string {
//	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
//	b := make([]byte, 8)
//	for i := range b {
//		b[i] = letters[rand.Intn(len(letters))]
//	}
//	return string(b)
//}
