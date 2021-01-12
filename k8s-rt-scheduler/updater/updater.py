#!/usr/bin/env python
from kubernetes import client, config
import os

config.load_kube_config("/var/run/kubernetes/admin.kubeconfig")
	
node_name = os.environ.get('MY_NODE_NAME', "127.0.0.1")

v1 = client.CoreV1Api()
print("Listing pods with their IPs on node: ", node_name)
field_selector = 'spec.nodeName='+str(node_name)
ret = v1.list_pod_for_all_namespaces(watch=False, field_selector=field_selector)



