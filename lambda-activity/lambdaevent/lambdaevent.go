package lambdaevent

import (
	"encoding/json"
	"fmt"
	"strings"
)

/*
This file contains functions and types to create data structures from lambda
function input received from lambda. You need to use the associated template
to map values into the structure
*/

type lambdaEvent struct {
	m map[string]string
}

// Decode takes raw json supplied to the lambda function via node.js and returns
// a struct with a GetValue func to extract data
func Decode(cmd string) (*lambdaEvent, error) {

	le := &lambdaEvent{}
	le.m = make(map[string]string)

	var err error
	var event map[string]json.RawMessage

	err = json.Unmarshal([]byte(cmd), &event)
	if err != nil {
		return le, fmt.Errorf("unable to find event data in input: %v\n", err)
	}

	for k, v := range event {
		if isJSON(v) {
			copyFromJSON(le, k, v)
		} else {
			le.m[strings.ToLower(k)] = string(v)
		}
	}
	return le, nil
}

// copyFromJSON recurcively copys elements from the source json string
func copyFromJSON(le *lambdaEvent, pk string, j json.RawMessage) {
	var tm map[string]json.RawMessage

	_ = json.Unmarshal(j, &tm)
	for k, v := range tm {
		if isJSON(v) {
			copyFromJSON(le, pk+"."+k, v)
		} else {
			le.m[strings.ToLower(pk+"."+k)] = string(v)
		}
	}
}

// isJSON returns true if it is given a json string
func isJSON(s json.RawMessage) bool {
	var js map[string]interface{}
	return json.Unmarshal(s, &js) == nil
}

// GetValue returns the event attribute as a string or an empty string if not found
func (le *lambdaEvent) GetValue(a string) string {
	s, _ := le.GetValueBool(a)
	return s
}

// GetValueBool returns the event attribute as a string and true or an empty string and false
// if the attribute can not be found
func (le *lambdaEvent) GetValueBool(a string) (string, bool) {

	// make sure the struct is valid before scanning it
	if le == nil {
		return "", false
	}

	for k, v := range le.m {
		if strings.EqualFold(a, k) {
			return strings.Trim(v, "\""), true
		}
	}
	return "", false
}

// ListAttributes prints out a list of all known attributes. Debugging
func (le *lambdaEvent) ListAttributes() {

	for k, v := range le.m {
		fmt.Printf("k: %s v: %s\n", k, v)
	}
}

// GetJSON will return a JSON encoded byte array of all values
func (le *lambdaEvent) GetJSON() ([]byte, error) {
	return json.MarshalIndent(le.m, "", "\t")
}

// GetKeys returns a slice of strings containing each available value name
func (le *lambdaEvent) GetKeys() []string {

	r := make([]string, len(le.m))
	for k, _ := range le.m {
		r = append(r, k)
	}
	return r
}

/*

 */
