# elasticipd
[![Build Status](https://github.com/syscll/elasticipd/workflows/ci/badge.svg)](https://github.com/syscll/elasticipd/actions) [![MIT licensed](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE) [![Quay.io](https://img.shields.io/badge/container-quay.io-red)](https://quay.io/repository/syscll/elasticipd)

As it is now common practice to run applications on top of a container-orchestration platforms, such as Kubernetes, there is no guarantee that a service will always run on the same host. This can cause problems when a service requiring a public IP address gets rescheduled.
`elasticipd` automatically associates an [Elastic IP](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/elastic-ip-addresses-eip.html) address to the AWS EC2 instance running this service. It is designed to run as a sidecar container alongisde a service that requires a public IP address.

## Usage
As `elasticipd` is currently configured to use AWS [Instance Roles](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html), the host will need to have and EC2 Policy with at least the following actions: `DisassociateAddress`, `DescribeInstanceAttribute`, `AssociateAddress` and `DisassociateAddress`.

The service is configured by setting the `ELASTIC_IP`, `AWS_REGION`, `POLL_INTERVAL` and `PORT` environment variables.

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
    image: quay.io/syscll/elasticipd:v1.3.0
    command:
    - elasticipd
    livenessProbe:
      httpGet:
        path: /healthz
        port: 8081
      initialDelaySeconds: 5
      periodSeconds: 3
    ports:
    - containerPort: 8081
    env:
    - name: ELASTIC_IP
      value: "192.168.0.1"
    - name: AWS_REGION
      value: "eu-west-1"
    - name: POLL_INTERVAL
      value: "10s"
```
