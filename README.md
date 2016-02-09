# aws-swf-lambda-example
Example Go code that runs in AWS Simple Workflow Service and Lambda functions


## Overview

This repo contains code I have developed while learning Amazon Web Services Simple Workflow Service. 

It contains a Go code decider that makes decisions based on business logic and a Go based Lambda function to simulate real world work taking place.

### Contents

**decider/**

This directory contains a Go language decider that polls the AWS SWF for any required decisions and then applies business logic to decide which tasks to run next.

**lambda-activity/**

This directory contains the code for a Go based Lambda function that simulates real world work


### Setup

To run the system you need to:
- Create an IAM role for the Lambda function. See the **iam-role.json** file under **lambda-activity/**
- Compile the Lambda code using the **buildeme.sh** script but do not upload to AWS yet
- Create the Lambda function as described at the top of the buildme.sh script and upload the zip file
- Compile the Go based decider in the **decider/** directory
- Update the **testit.sh** script to use the correct ARN for the Lambda role you created
- Run the **testit.sh** script and it will start a new SWF workflow then run the decider to process the tasks
- Note that the Lambda function has a randomFail function that inserts random fails into the system to simulate failures

**NOTE** - This repo contains Go language code but you can not use go get ... to get the code


