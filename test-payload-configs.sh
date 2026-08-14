#!/bin/bash
# QE OpenSearch
source ~/.creds/DANGER_LOCAL_QE_TEST_ELK.creds

# perfscale intlab
#source ~/.creds/DANGER_LOCAL_INTLAB_OPENSEARCH.creds

set -x
LOOKBACK=15d
DEBUG=--debug
DISPLAY="--display buildTag,masterNodesType,workerNodesCount,fips,benchmark,buildUrl"
CONFIG=examples/node-density.sbdb-shift-detect.yaml

#for VERSION in 4.22 ; do
for VERSION in 5.0 ; do
  # payload input vars
  read -r -d '' INPUT_VARS << EOF
{
  "distribution": "openshift",
  "stream": "ocp",
  "microshift": false,
  "platform": "AWS",
  "clusterType": "self-managed",
  "ocpMajorVersion": "${VERSION}",
  "masterNodesType": "m6a.xlarge",
  "workerNodesType": "m6a.xlarge",
  "masterNodesCount": 3,
  "infraNodesType": "r5.xlarge",
  "workerNodesCount": 6,
  "infraNodesCount": 3,
  "totalNodes": 12,
  "sdnType": "OVNKubernetes",
  "region": "us-west-2",
  "fips": false,
  "publish": "External",
  "workerArch": "amd64",
  "controlPlaneArch": "amd64",
  "ipsec": false,
  "ipsecMode": "Disabled"
}
EOF

  # 2. Compact it into a single line
  CONDENSED_JSON=$(echo "$INPUT_VARS" | jq -c .)
  NOW=$(date +"%F-%H%M%S")
  OUTFILE=$( basename $0 ).${VERSION}.${NOW}.log
VERSION=$VERSION orion --config $CONFIG --jira-ack --lookback 15d --hunter-analyze --input-vars="$CONDENSED_JSON" --output-format junit --save-output-path=junit.xml --github-repos kube-burner/kube-burner,kube-burner/kube-burner-ocp,cloud-bulldozer/e2e-benchmarking,openshift/release | tee -a $OUTFILE
done
