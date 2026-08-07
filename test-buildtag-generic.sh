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

orion $DEBUG $DISPLAY --hunter-analyze --build-tag 2085683329012600832 --config examples/generic.yaml
