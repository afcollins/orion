#!/bin/bash
# QE OpenSearch
source ~/.creds/DANGER_LOCAL_QE_TEST_ELK.creds

# perfscale intlab
#source ~/.creds/DANGER_LOCAL_INTLAB_OPENSEARCH.creds

set -x
LOOKBACK=15d
DEBUG=--debug
DISPLAY="--display buildTag,masterNodesType,workerNodesCount,fips,benchmark,buildUrl"

for VERSION in 4.22 ; do
#for VERSION in 5.0 ; do
  NOW=$(date +"%F-%H%M%S")
  OUTFILE=$( basename $0 ).${VERSION}.${NOW}.log
VERSION=$VERSION orion --config examples/cluster-density.yaml --jira-ack --lookback 15d --hunter-analyze '--input-vars={"distribution":"openshift","stream":"ocp","microshift":false,"platform":"AWS","clusterType":"self-managed","ocpVersion":"4.22.0-0.nightly-2026-08-12-143251","ocpMajorVersion":"4.22","k8sVersion":"v1.35.6","masterNodesType":"m6a.xlarge","workerNodesType":"m6a.xlarge","masterNodesCount":3,"infraNodesType":"r5.xlarge","workerNodesCount":6,"infraNodesCount":3,"totalNodes":12,"sdnType":"OVNKubernetes","clusterName":"ci-op-x8759rrx-1e4cf-kt8h4","region":"us-west-2","fips":false,"publish":"External","workerArch":"amd64","controlPlaneArch":"amd64","ipsec":false,"ipsecMode":"Disabled"}' --output-format junit --save-output-path=junit.xml --github-repos kube-burner/kube-burner,kube-burner/kube-burner-ocp,cloud-bulldozer/e2e-benchmarking,openshift/release | tee -a $OUTFILE
done
