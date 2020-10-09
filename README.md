# elasticipd [![Build Status](https://travis-ci.org/syscll/elasticipd.svg?branch=master)](https://travis-ci.org/syscll/elasticipd)

As it is now common practice to run applications on top of a container-orchestration platforms, such as Kubernetes, there is no guarantee that a service will always run on the same host. This can cause problems when a service requiring a public IP address gets rescheduled.
`elasticipd` automatically associates an [Elastic IP](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/elastic-ip-addresses-eip.html) address to the AWS EC2 instance running this service. It is designed to run as a sidecar container alongisde a service that requires a public IP address.

## Usage
As `elasticipd` is currently configured to use AWS [Instance Roles](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html), the host will need to have and EC2 Policy with at least the following actions: `DisassociateAddress`, `DescribeInstanceAttribute`, `AssociateAddress` and `DisassociateAddress`.

The service is configured by setting the `ELASTIC_IP` and `AWS_REGION` environment variables.

### Kubernetes
A simple multi-container Pod spec:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: public-service
spec:
  securityContext:
    runAsUser: 2000
  containers:
  - name: public-service
    image: public-service
  - name: elasticipd
    image: <REPO>/elasticipd
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
      value: "1.1.1.1"
    - name: AWS_REGION
      value: "eu-west-1"
```
