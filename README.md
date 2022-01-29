# elasticipd
[![Build Status](https://github.com/syscll/elasticipd/workflows/ci/badge.svg)](https://github.com/syscll/elasticipd/actions) [![MIT licensed](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE) [![Quay.io](https://img.shields.io/badge/container-quay.io-red)](https://quay.io/repository/syscll/elasticipd)

As it is now common practice to run applications on top of a container-orchestration platforms, such as Kubernetes, there is no guarantee that a service will always run on the same host. This can cause problems when a service requiring a public IP address gets rescheduled.
`elasticipd` automatically associates an [Elastic IP](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/elastic-ip-addresses-eip.html) address to the AWS EC2 instance running this service. It is designed to run as a sidecar container alongisde a service that requires a public IP address.

## Usage
As `elasticipd` is currently configured to use AWS [Instance Roles](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html), the host will need to have and EC2 Policy with at least the following actions: `DescribeInstances`, `DescribeInstanceAttribute`, `AssociateAddress` and `DisassociateAddress`.

The service is configured by setting the following command line flags:
```
Usage of elasticipd:
  -elastic-ip string
        Elastic IP address to associate
  -interval string
        Attempt association every interval (default "30s")
  -port int
        Local HTTP server port (default 8081)
  -reassoc
        Allow Elastic IP to be reassociated without failure (default true)
  -region string
        AWS region hosting the Elastic IP and EC2 instance
  -retries int
        Maximum number of association retries before fatally exiting (default 3)
```

### Kubernetes
A simple multi-container Pod spec:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: public-service
spec:
  containers:
  - name: public-service
    image: public-service
  - name: elasticipd
    image: quay.io/syscll/elasticipd:v2.2.0
    command: ["elasticipd"]
    args: [
        "-elastic-ip=<elastic_ip>",
        "-region=us-west-2",
        "-interval=10s"
    ]
    ports:
    - containerPort: 8081
      name: local-http
    livenessProbe:
      httpGet:
        path: /healthz
        port: local-http
      initialDelaySeconds: 3
      periodSeconds: 3
```
