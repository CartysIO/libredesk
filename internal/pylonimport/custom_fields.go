package pylonimport

import (
	"strings"
)

// MergeIssueCustomAttributes copies every Pylon custom field into Libredesk conversation custom_attributes.
// Colliding keys get a pylon. prefix.
func MergeIssueCustomAttributes(existing map[string]string, fields map[string]CustomField) map[string]string {
	out := map[string]string{}
	for k, v := range existing {
		out[k] = v
	}
	for key, f := range fields {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		val := customFieldText(f)
		if val == "" {
			continue
		}
		dest := key
		if _, collides := out[dest]; collides {
			dest = "pylon." + key
		}
		if _, collides := out[dest]; collides {
			continue
		}
		out[dest] = val
	}
	return out
}

func customFieldText(f CustomField) string {
	var vals []string
	for _, v := range append([]string{f.Value}, f.Values...) {
		v = strings.TrimSpace(v)
		if v != "" {
			vals = append(vals, v)
		}
	}
	return strings.Join(vals, ", ")
}