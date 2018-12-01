# elasticipd [![Build Status](https://travis-ci.org/syscll/elasticipd.svg?branch=master)](https://travis-ci.org/syscll/elasticipd)

As it is now common practice to run applications on top of a container-orchestration system, such as Kubernetes, there is no guarantee that a servive will always run on the same host. This can cause problems when a service requiring a public IP address gets rescheduled.

`elasticipd` automatically associates an [Elastic IP](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/elastic-ip-addresses-eip.html) address to the AWS EC2 instance running this service. It is designed to run as a sidecar container alongisde a service that requires a public IP address.

## Usage

As `elasticipd` is currently configured to use AWS [Instance Roles](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html), the host will need to have `AmazonEC2FullAccess`.

The Elastic IP address is configured by setting the `ELASTIC_IP` environment variable.

### Kubernetes

A simple multi-container Pod spec:

```
apiVersion: v1
kind: Pod
metadata:
  name: public-service
spec:
  containers:
  - name: public-service
    image: public-service
  - name: elasticipd
    image: danbondd/elasticipd:latest
    command:
    - elasticipd
    env:
    - name: ELASTIC_IP
      valueFrom:
        configMapKeyRef:
          name: public-service
          key: elastic-ip
```
