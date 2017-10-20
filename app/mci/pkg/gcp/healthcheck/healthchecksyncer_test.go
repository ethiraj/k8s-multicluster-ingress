// Copyright 2017 Google Inc.
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

package healthcheck

import (
	"net/http"
	"testing"

	compute "google.golang.org/api/compute/v1"

	ingresshc "k8s.io/ingress-gce/pkg/healthchecks"
	"k8s.io/ingress-gce/pkg/utils"

	utilsnamer "github.com/GoogleCloudPlatform/k8s-multicluster-ingress/app/mci/pkg/gcp/namer"
	sp "github.com/GoogleCloudPlatform/k8s-multicluster-ingress/app/mci/pkg/serviceport"
)

func TestEnsureHealthCheck(t *testing.T) {
	lbName := "lb-name"
	port := int64(32211)
	// Should create the health check as expected.
	hcp := ingresshc.NewFakeHealthCheckProvider()
	namer := utilsnamer.NewNamer("mci", lbName)
	hcName := namer.HealthCheckName(port)
	hcs := NewHealthCheckSyncer(namer, hcp)

	testCases := []struct {
		// Human-readable description of test.
		desc string
		// Inputs
		protocol    string
		forceUpdate bool
		// Outputs
		ensureErr bool
	}{
		{
			desc:        "expected no error in ensuring health check",
			protocol:    "HTTP",
			forceUpdate: false,
			ensureErr:   false,
		},
		{
			desc:        "writing same health check should not error (forceUpdate=false)",
			protocol:    "HTTP",
			forceUpdate: false,
			ensureErr:   false,
		},
		{
			desc:        "writing same health check should not error (forceUpdate=true)",
			protocol:    "HTTP",
			forceUpdate: true,
			ensureErr:   false,
		},
		{
			desc:        "a different healthcheck should cause an error when forceUpdate=false",
			protocol:    "HTTPS", /* Not the original HTTP */
			forceUpdate: false,
			ensureErr:   true,
		},
		{
			desc:        "a different healthcheck should not error when forceUpdate=true",
			protocol:    "HTTPS", /* Not the original HTTP */
			forceUpdate: true,
			ensureErr:   false,
		},
	}

	// GET should return NotFound.
	if _, err := hcp.GetHealthCheck(hcName); !utils.IsHTTPErrorCode(err, http.StatusNotFound) {
		t.Fatalf("expected NotFound error before EnsureHealthCheck, actual: %v", err)
	}

	for _, c := range testCases {
		err := hcs.EnsureHealthCheck(lbName, []sp.ServicePort{
			{
				Port:     port,
				Protocol: c.protocol,
			},
		}, c.forceUpdate)
		if (err != nil) != c.ensureErr {
			t.Errorf("error when: %v: EnsureHealthCheck({%v,%v}, %v) = [%v]. Want err? %t.",
				c.desc, port, c.protocol, c.forceUpdate, err, c.ensureErr)
		}
		// Verify that GET does not return NotFound.
		if _, err := hcp.GetHealthCheck(hcName); err != nil {
			t.Fatalf("expected nil error, actual: %v", err)
		}

	}

	// TODO(G-Harmon): Validate values in health check.
}

func TestHealthCheckMatches(t *testing.T) {
	var check compute.HealthCheck
	if !healthCheckMatches(&check, &check) {
		t.Errorf("Want healthCheckMatches(c, c) = true. got false.")
	}
	check2 := check
	check2.Description = "foo"
	if healthCheckMatches(&check, &check2) {
		t.Errorf("Want healthCheckMatches(c, c2) = false, c.description differs. got true.")
	}
	check.Description = "foo"
	if !healthCheckMatches(&check, &check2) {
		t.Errorf("Health checks should be identical again c:%v, c2:%v", check, check2)
	}
	// CreationTimestamp should be ignored.
	check2.CreationTimestamp = "1234"
	if !healthCheckMatches(&check, &check2) {
		t.Errorf("Health checks only differ in creation timestamp, watch Matches(c, c2)=true, got false")
	}
}
