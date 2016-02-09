package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/swf"
)

type decision struct {
	svc *swf.SWF
	tt  string // task token associated with this decision
}

type LambdaInput struct { // send to lambda worker to ask it to work
	ReqType  string // type of request to run
	ReqInput string // input data for request
}

type activity struct {
	svc        *swf.SWF // AWS access structure
	tt         string   // TaskToken
	id         string   // activity id
	name       string   // activity type name
	version    string   // activity type version
	input      string   // user defined input string
	context    string   // user defined string
	function   string   // Lambda Function name
	stcTimeout string   // StartToClose timeout for Lambda function
}

type ActivityResult struct {
	ResultType   string // type that can be used by the deciders for next steps
	ResultOutput string // any output produced during the work - MUST be a json string if going to Lambda function
	ResultErr    string // any error messages produced during the work
}

func main() {

	// set up some values to help with running examples
	swfDomain := "swfGoTest"
	swfTasklist := "swfGoTesttln"
	swfIdentity := "gocode-decider"

	swfsvc := swf.New(session.New())

	params := &swf.PollForDecisionTaskInput{
		Domain: aws.String(swfDomain), //
		TaskList: &swf.TaskList{ //
			Name: aws.String(swfTasklist), //
		},
		Identity:        aws.String(swfIdentity),
		MaximumPageSize: aws.Int64(100),
		ReverseOrder:    aws.Bool(true),
	}

	// loop forever while polling for work
	for {
		resp, err := swfsvc.PollForDecisionTask(params)
		if err != nil {
			log.Fatalf("error: unable to poll for decision: %v\n", err)
		}

		// if we do not receive a task token then 60 second time out occured so try again
		if resp.TaskToken != nil {
			if *resp.TaskToken != "" {
				d := &decision{
					svc: swfsvc,
					tt:  *resp.TaskToken,
				}
				// make each decision in a goroutine which means that multiple
				// decisions may be running at the same time
				go d.makeDecision(resp.Events)
			}
		} else {
			log.Printf("debug - no decisions required\n")
		}
	}

}

func (d *decision) makeDecision(events []*swf.HistoryEvent) {

	fmt.Printf("\n#################### handle New Decision\n%s\n####################\n", d.tt)

	var handled bool
	var err error

	// loop backwards through time and make decisions
	for k, event := range events {
		switch *event.EventType {
		case "WorkflowExecutionStarted":
			// new work flow so start the first activity
			fmt.Printf("debug handling eventType: %s\n", *event.EventType)
			err = d.handleNewWorkFlow(event)
			handled = true
		case "LambdaFunctionCompleted":
			// latest event was lambda task completed
			fmt.Printf("debug handling eventType: %s\n", *event.EventType)
			err = d.handleLambdaComplete(k, events)
			handled = true
		case "TimerFired":
			fmt.Printf("debug handling eventType: %s\n", *event.EventType)
			err = d.handleTimerFired(k, events)
			handled = true
		default:
			fmt.Printf("debug unhandled eventType: %s\n", *event.EventType)
		}
		if handled == true {
			break // decision has been made so stop scanning the events
		}
	}

	if err != nil {
		fmt.Printf("error making decision. workflow failed: %v\n", err)
		// we are not able to process the workflow so fail it
		err2 := d.failWorkflow("", err)
		if err2 != nil {
			fmt.Printf("error while failing workflow: %v\n", err2)
		}
	}

	if handled == false {
		fmt.Printf("debug dump of received event for taskToken: %s\n", d.tt)
		fmt.Println(events)
		fmt.Printf("xxxx debug unhandled decision\n")
	}
	fmt.Printf("\n#################### completed handling Decision\n%s\n####################\n", d.tt)
	// exit goroutine
}

// businessLogic makes the actual decisions and starts the next step
func (d *decision) businessLogic(ar *ActivityResult) error {
	var err error

	// switch depending on input data
	switch ar.ResultType {
	case "newworkflow":
		// start of new workflow event so setup next activity
		na := &activity{
			svc:        d.svc,
			tt:         d.tt,
			id:         "amicreate",             // activity id
			function:   "swfec2ssworker",        // lambda function to call
			input:      jsonIt("amicreate", ""), // input to pass to lambda function
			stcTimeout: "10",                    // timeout timer
			context:    "swfGoTest-context",     // context
		}
		err = d.scheduleNextLambda(na)
	case "amicreate":
		// 17 second pause after amicreate then tag the ami's
		err = d.setTimer("17", ar.ResultOutput, "amicreate-timer")
		if err != nil {
			fmt.Printf("debug err from set timer: %v\n", err)
		}
	case "amicreate-timer":
		// amicreate-timer done so tag the new ami's so next step is tagandremoveold
		na := &activity{
			svc:        d.svc,
			tt:         d.tt,
			id:         "tagami",
			function:   "swfec2ssworker",
			input:      jsonIt("tagami", ar.ResultOutput),
			stcTimeout: "10", // timeout timer
			context:    "swfGoTest-context",
		}
		err = d.scheduleNextLambda(na)
	case "tagami":
		// new ami's have been tagged so remove old ami's
		na := &activity{
			svc:        d.svc,
			tt:         d.tt,
			id:         "removeold",
			function:   "swfec2ssworker",
			input:      jsonIt("removeold", ar.ResultOutput),
			stcTimeout: "10", // timeout timer
			context:    "swfGoTest-context",
		}
		err = d.scheduleNextLambda(na)
	case "removeold":
		// 12 second pause after old ami's remove before deleting snapshots
		err = d.setTimer("12", ar.ResultOutput, "removeold-timer")
		if err != nil {
			fmt.Printf("debug err from set timer: %v\n", err)
		}
	case "removeold-timer":
		// remove old ami's from system
		na := &activity{
			svc:        d.svc,
			tt:         d.tt,
			id:         "deletesnapshots",
			function:   "swfec2ssworker",
			input:      jsonIt("deletesnapshots", ar.ResultOutput),
			stcTimeout: "10", // timeout timer
			context:    "swfGoTest-context",
		}
		err = d.scheduleNextLambda(na)
	case "deletesnapshots":
		// all tasks done so advise SWF we are complete
		err = d.completeWorkflow("All done")
		fmt.Printf("++++ completed workflow\n")
	default:
		err = fmt.Errorf("error - unknown inputs for business logic: %s\n", ar.ResultType)
	}
	return err
}

func (d *decision) handleNewWorkFlow(e *swf.HistoryEvent) error {

	ar := &ActivityResult{
		ResultType:   "newworkflow",                          //
		ResultOutput: "{\"startnewworkflow\":\"no inputs\"}", // constraint means must have some test
		ResultErr:    "",
	}
	fmt.Printf("activity result calling business logic from new work flow\n")
	fmt.Println(ar)
	return d.businessLogic(ar)
}

func (d *decision) handleTimerFired(k int, es []*swf.HistoryEvent) error {

	// ok we have an timer fired so get the starteventid and find the data associated with it
	evId := es[k].TimerFiredEventAttributes.StartedEventId
	timerId := es[k].TimerFiredEventAttributes.TimerId

	var result string

	for _, ev := range es {
		if *ev.EventId == *evId {
			// found the timer start which has the control field
			// which holds the result from the activity before the timer
			result = *ev.TimerStartedEventAttributes.Control
			break
		}
	}

	ar := &ActivityResult{
		ResultType:   *timerId, // eor example amicreate-timer so business logic can make decisions
		ResultOutput: result,   // the control field of the event that started the timer
		ResultErr:    "",
	}
	// extract the data, create an activity result struct and pass it to bubiness logic

	fmt.Printf("activity result calling business logic from timer fired\n")
	fmt.Println(ar)
	// decide next steps
	return d.businessLogic(ar)
}

func (d *decision) handleLambdaComplete(k int, es []*swf.HistoryEvent) error {

	var ar ActivityResult

	t, err := strconv.Unquote(*es[k].LambdaFunctionCompletedEventAttributes.Result)

	err = json.Unmarshal([]byte(t), &ar)
	if err != nil {
		return fmt.Errorf("error with json unmarshal: %v\n", err)
	}

	if ar.ResultErr != "" {
		return fmt.Errorf("error received from Lambda worker. type: %s err: %s output: %s\n", ar.ResultType, ar.ResultErr, ar.ResultOutput)
	}

	fmt.Printf("activity result calling business logic from lambda complete\n")
	fmt.Println(ar)

	// decide next steps
	return d.businessLogic(&ar)
}

func (d *decision) setTimer(sec, data, id string) error {
	fmt.Printf("debug start set timer to wait: %s seconds\n", sec)

	params := &swf.RespondDecisionTaskCompletedInput{
		TaskToken: aws.String(d.tt),
		Decisions: []*swf.Decision{
			{
				DecisionType: aws.String("StartTimer"),
				StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
					StartToFireTimeout: aws.String(sec),
					TimerId:            aws.String(id),
					Control:            aws.String(data),
				},
			},
		},
		ExecutionContext: aws.String("ssec2-amicreate"),
	}
	_, err := d.svc.RespondDecisionTaskCompleted(params)
	return err

}

func (d *decision) completeWorkflow(result string) error {

	params := &swf.RespondDecisionTaskCompletedInput{
		TaskToken: aws.String(d.tt),
		Decisions: []*swf.Decision{
			{
				DecisionType: aws.String("CompleteWorkflowExecution"),
				CompleteWorkflowExecutionDecisionAttributes: &swf.CompleteWorkflowExecutionDecisionAttributes{
					Result: aws.String(result),
				},
			},
		},
		ExecutionContext: aws.String("Data"),
	}
	_, err := d.svc.RespondDecisionTaskCompleted(params)
	return err // which may be nil
}

func (d *decision) failWorkflow(details string, err error) error {

	params := &swf.RespondDecisionTaskCompletedInput{
		TaskToken: aws.String(d.tt),
		Decisions: []*swf.Decision{
			{
				DecisionType: aws.String("FailWorkflowExecution"),
				FailWorkflowExecutionDecisionAttributes: &swf.FailWorkflowExecutionDecisionAttributes{
					Details: aws.String(details),
					Reason:  aws.String(fmt.Sprintf("%v", err)),
				},
			},
		},
		ExecutionContext: aws.String("Data"),
	}
	_, err = d.svc.RespondDecisionTaskCompleted(params)
	return err // which may be nil
}

func (d *decision) scheduleNextLambda(na *activity) error {
	params := &swf.RespondDecisionTaskCompletedInput{
		TaskToken: aws.String(na.tt),
		Decisions: []*swf.Decision{
			{
				DecisionType: aws.String("ScheduleLambdaFunction"), //
				ScheduleLambdaFunctionDecisionAttributes: &swf.ScheduleLambdaFunctionDecisionAttributes{
					Id:                  aws.String(na.id),       //
					Name:                aws.String(na.function), //
					Input:               aws.String(na.input),
					StartToCloseTimeout: aws.String(na.stcTimeout),
				},
			},
		},
		ExecutionContext: aws.String(na.context),
	}
	_, err := d.svc.RespondDecisionTaskCompleted(params)
	return err
}

// jsonIt takes t2 inputs and returns a json string containing them
func jsonIt(reqType, reqInput string) string {

	li := &LambdaInput{
		ReqType:  reqType,
		ReqInput: reqInput,
	}
	b, err := json.Marshal(li)
	if err != nil {
		return ""
	}
	return string(b)
}

/*

 */
