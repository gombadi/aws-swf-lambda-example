/*

This code is called as a Lambda function in response to AWS Simple Workflow decisions.
This one function can handle multiple work activities which has the benifit that
the lambda function code will be run multiple times and therefore be kept in the system.

*/
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/gombadi/lambda-snapshot/lambdaevent"
)

// json encoded result sent back to SWF deciders
type LambdaResult struct {
	ResultType   string // type that can be used by the deciders for next steps
	ResultOutput string // any output produced during the work
	ResultErr    string // any error messages produced during the work
}

func main() {

	var result []byte

	// decode the last commandline arg which is the SWF Input structure
	le, err := lambdaevent.Decode(os.Args[len(os.Args)-1])
	if err != nil {
		log.Fatalf("error: unable to create new lambda event: %v\n", err)
	}

	reqType := le.GetValue("reqtype")
	reqInput := le.GetValue("reqinput")

	switch reqType {
	case "amicreate":
		result, err = startAMICreate(reqInput)
	case "tagami":
		result, err = startTagAMI(reqInput)
	case "removeold":
		result, err = startRemoveOld(reqInput)
	case "deletesnapshots":
		result, err = startDeleteSnapshots(reqInput)
	default:
		lr := &LambdaResult{
			ResultType:   "",
			ResultOutput: "",
			ResultErr:    fmt.Sprintf("error: unknown request type: %s input: %s\n", reqType, reqInput),
		}
		result, err = json.Marshal(lr)
	}
	if err != nil {
		log.Fatalf("error with json.Marshal of result. unable to send reply: %v\n", err)
	}

	// all lambda worker needs to do is send back a result and wrapper/lambda/SWF will pass to decider
	fmt.Printf("%s", string(result))
}

// randomFails will return an error message about 50% of the time to test
// decider failure paths
func randomFails() string {

	rand.Seed(time.Now().Unix())
	if x := rand.Intn(10); x >= 6 {
		return "A random error occured while processing the request"
	}
	return ""
}

// The following functions are example stubs for the different tasks that
// this Lambda function can perform

func startAMICreate(reqInput string) ([]byte, error) {
	log.Printf("debug - start simulated ami create with input: %s\n", reqInput)
	log.Printf("####################################\n")
	lr := &LambdaResult{
		ResultType:   "amicreate",
		ResultOutput: "ami-abcd1234,ami-1234abcd",
		ResultErr:    randomFails(),
	}
	return json.Marshal(lr)
}

func startTagAMI(reqInput string) ([]byte, error) {
	log.Printf("debug - start simulated tag ami with input: %s\n", reqInput)
	log.Printf("####################################\n")
	lr := &LambdaResult{
		ResultType:   "tagami",
		ResultOutput: "",
		ResultErr:    randomFails(),
	}
	return json.Marshal(lr)
}

func startRemoveOld(reqInput string) ([]byte, error) {
	log.Printf("debug - start simulated remove old ami with input: %s\n", reqInput)
	log.Printf("####################################\n")
	lr := &LambdaResult{
		ResultType:   "removeold",
		ResultOutput: "snap-efgh5678,snap-5678efgh",
		ResultErr:    randomFails(),
	}
	return json.Marshal(lr)
}

func startDeleteSnapshots(reqInput string) ([]byte, error) {
	log.Printf("debug - start simulated delete of old snapshots with input: %s\n", reqInput)
	log.Printf("####################################\n")
	lr := &LambdaResult{
		ResultType:   "deletesnapshots",
		ResultOutput: "",
		ResultErr:    "",
	}
	return json.Marshal(lr)
} /*

 */
