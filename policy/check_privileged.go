/*
Copyright 2021 The Kubernetes Authors.

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

package policy

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/pod-security-admission/api"
)

/*
Privileged Pods disable most security mechanisms and must be disallowed.

Restricted Fields:
spec.containers[*].securityContext.privileged
spec.initContainers[*].securityContext.privileged

Allowed Values: false, undefined/null
*/

func init() {
	addCheck(CheckPrivileged)
}

// CheckPrivileged returns a baseline level check
// that forbids privileged=true in 1.0+
func CheckPrivileged() Check {
	return Check{
		ID:    "privileged",
		Level: api.LevelBaseline,
		Versions: []VersionedCheck{
			{
				MinimumVersion: api.MajorMinorVersion(1, 0),
				CheckPod:       withOptions(privilegedV1Dot0),
			},
		},
	}
}

func privilegedV1Dot0(podMetadata *metav1.ObjectMeta, podSpec *corev1.PodSpec, opts options) CheckResult {
	badContainers := NewViolations(opts.withFieldErrors)

	visitContainers(podSpec, opts, func(container *corev1.Container, path *field.Path) {
		if container.SecurityContext != nil && container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
			if opts.withFieldErrors {
				badContainers.Add(container.Name, withBadValue(forbidden(path.Child("securityContext", "privileged")), true))
			} else {
				badContainers.Add(container.Name)
			}
		}
	})

	if !badContainers.Empty() {
		return CheckResult{
			Allowed:         false,
			ForbiddenReason: "privileged",
			ForbiddenDetail: fmt.Sprintf(
				`%s %s must not set securityContext.privileged=true`,
				pluralize("container", "containers", badContainers.Len()),
				joinQuote(badContainers.Data()),
			),
			ErrList: badContainers.Errs(),
		}
	}
	return CheckResult{Allowed: true}
}
