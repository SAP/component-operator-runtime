/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package templatex

import "bytes"

// This function mimics the Helm behaviour. Background: values passed to the built-in generators
// are of type map[string]any. Of course, templates are rendered with the missingkey=zero option.
// But still, if a key is missing in the values, the empty value of 'any' (returned in this case)
// makes the go templating engine return '<no value>' in that case.
// Helm decided to override that by replacing all occurrences of the string '<no value>' in any template output
// by the empty string. We are following that approach.
func AdjustTemplateOutput(data []byte) []byte {
	return bytes.ReplaceAll(data, []byte("<no value>"), []byte(""))
}
