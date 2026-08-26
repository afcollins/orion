#!/bin/bash
set -x
#SCRIPTNAME=$( basename $0 )
# QE OpenSearch
source ~/.creds/DANGER_LOCAL_QE_TEST_ELK.creds

echo 
# perfscale intlab
#source ~/.creds/DANGER_LOCAL_INTLAB_OPENSEARCH.creds

DEBUG=--debug
DISPLAY="--display buildTag,masterNodesType,workerNodesCount,fips,benchmark,buildUrl"

uv run main.py $DEBUG $DISPLAY --hunter-analyze --build-tag 2089779365775675392 --build-tag 2090401610168537088 --build-tag 2090481448787120128 --build-tag 2089779365834395648 --build-tag 2090401610235645952 --build-tag 2090481451307896832 --build-tag 2090481455502200832 --config examples/generic.yaml
