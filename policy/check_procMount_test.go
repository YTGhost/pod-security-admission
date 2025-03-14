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
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestProcMount(t *testing.T) {
	defaultValue := corev1.DefaultProcMount
	unmaskedValue := corev1.UnmaskedProcMount
	otherValue := corev1.ProcMountType("other")

	hostUsers := false
	tests := []struct {
		name           string
		pod            *corev1.Pod
		opts           options
		expectReason   string
		expectDetail   string
		expectErrList  field.ErrorList
		expectAllowed  bool
		relaxForUserNS bool
	}{
		{
			name: "procMount",
			pod: &corev1.Pod{Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "a", SecurityContext: nil},
					{Name: "b", SecurityContext: &corev1.SecurityContext{}},
					{Name: "c", SecurityContext: &corev1.SecurityContext{ProcMount: &defaultValue}},
					{Name: "d", SecurityContext: &corev1.SecurityContext{ProcMount: &unmaskedValue}},
					{Name: "e", SecurityContext: &corev1.SecurityContext{ProcMount: &otherValue}},
				},
				HostUsers: &hostUsers,
			}},
			expectReason:  `procMount`,
			expectAllowed: false,
			expectDetail:  `containers "d", "e" must not set securityContext.procMount to "Unmasked", "other"`,
		},
		{
			name: "procMount",
			pod: &corev1.Pod{Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "a", SecurityContext: nil},
					{Name: "b", SecurityContext: &corev1.SecurityContext{}},
					{Name: "c", SecurityContext: &corev1.SecurityContext{ProcMount: &defaultValue}},
					{Name: "d", SecurityContext: &corev1.SecurityContext{ProcMount: &unmaskedValue}},
					{Name: "e", SecurityContext: &corev1.SecurityContext{ProcMount: &otherValue}},
				},
				HostUsers: &hostUsers,
			}},
			expectReason:   "",
			expectDetail:   "",
			expectAllowed:  true,
			relaxForUserNS: true,
		},
		{
			name: "procMount, enable field error list",
			pod: &corev1.Pod{Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "a", SecurityContext: nil},
					{Name: "b", SecurityContext: &corev1.SecurityContext{}},
					{Name: "c", SecurityContext: &corev1.SecurityContext{ProcMount: &defaultValue}},
					{Name: "d", SecurityContext: &corev1.SecurityContext{ProcMount: &unmaskedValue}},
					{Name: "e", SecurityContext: &corev1.SecurityContext{ProcMount: &otherValue}},
				},
			}},
			opts: options{
				withFieldErrors: true,
			},
			expectReason: `procMount`,
			expectDetail: `containers "d", "e" must not set securityContext.procMount to "Unmasked", "other"`,
			expectErrList: field.ErrorList{
				{Type: field.ErrorTypeForbidden, Field: "spec.containers[3].securityContext.procMount", BadValue: "Unmasked"},
				{Type: field.ErrorTypeForbidden, Field: "spec.containers[4].securityContext.procMount", BadValue: "other"},
			},
		},
	}

	cmpOpts := []cmp.Option{cmpopts.IgnoreFields(field.Error{}, "Detail"), cmpopts.SortSlices(func(a, b *field.Error) bool { return a.Error() < b.Error() })}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.relaxForUserNS {
				RelaxPolicyForUserNamespacePods(true)
				t.Cleanup(func() {
					RelaxPolicyForUserNamespacePods(false)
				})
			}
			result := procMountV1Dot0(&tc.pod.ObjectMeta, &tc.pod.Spec, tc.opts)
			if result.Allowed != tc.expectAllowed {
				t.Fatalf("expected Allowed to be %v was %v", tc.expectAllowed, result.Allowed)
			}
			if e, a := tc.expectReason, result.ForbiddenReason; e != a {
				t.Errorf("expected\n%s\ngot\n%s", e, a)
			}
			if e, a := tc.expectDetail, result.ForbiddenDetail; e != a {
				t.Errorf("expected\n%s\ngot\n%s", e, a)
			}
			if result.ErrList != nil {
				if diff := cmp.Diff(tc.expectErrList, *result.ErrList, cmpOpts...); diff != "" {
					t.Errorf("unexpected field errors (-want,+got):\n%s", diff)
				}
			}
		})
	}
}
