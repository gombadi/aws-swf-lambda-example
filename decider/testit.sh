#!/bin/bash

# This is a short script to start a workflow execution running then 
# start the decider to manage the workflow

echo "Starting workflow execution in AWS"
aws swf start-workflow-execution \
    --domain swfGoTest \
    --workflow-type name="swfGoTesttln",version="v1.0" \
    --lambda-role "arn:aws:iam::123456789876:role/swf-lambda-role" \
    --workflow-id wfid-$(date +%s)

echo
echo "Now start the decicer to process the new workflow"
echo
echo "Press CTRL-C to exit the deicder"
./lambda-decider
