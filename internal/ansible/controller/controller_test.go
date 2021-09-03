// Copyright 2021 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFilterPredicate(t *testing.T) {
	matchLabelPass := make(map[string]string)
	matchLabelPass["testKey"] = "testValue"
	selectorPass := metav1.LabelSelector{
		MatchLabels: matchLabelPass,
	}
	noSelector := metav1.LabelSelector{}

	passPredicate, err := parsePredicateSelector(selectorPass)
	assert.Equal(t, nil, err, "Verify that no error is thrown on a valid populated selector")
	assert.NotEqual(t, nil, passPredicate, "Verify that a predicate is constructed using a valid selector")

	nilPredicate, err := parsePredicateSelector(noSelector)
	assert.Equal(t, nil, err, "Verify that no error is thrown on a valid unpopulated selector")
	assert.Equal(t, nil, nilPredicate, "Verify correct parsing of an unpopulated selector")
}
