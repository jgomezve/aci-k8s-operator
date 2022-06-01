/*
Copyright 2022.

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

package controllers

import (
	"strconv"
	"strings"

	"github.com/jgomezve/aci-k8s-operator/api/v1alpha1"
)

func flattenRules(rules []v1alpha1.RuleSpec) string {

	listRules := []string{}
	for _, rule := range rules {
		item := rule.Eth
		if rule.IP != "" {
			item = item + "-" + rule.IP
		}
		if rule.Port != 0 {
			item = item + "-" + strconv.Itoa(rule.Port)
		}
		listRules = append(listRules, item)
	}
	return strings.Join(listRules, ", ")
}
