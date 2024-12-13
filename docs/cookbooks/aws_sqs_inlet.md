# AWS SQS Inlet

## Prerequisites
### Synadia Cloud
For the time being, Connectors are only available through Synadia Cloud. As such, you will need to have a Synadia Cloud
account and have the connectors service enabled on your account.

Within Synadia Cloud, navigate to the `Settings` page within your account. You will see a toggle switch to enable the
service.

### AWS SQS
Setting up SQS is not part of this getting started guide. Take a look at the
[SQS getting started guide](https://aws.amazon.com/sqs/getting-started/) to set up a queue. Make sure to take note of
the queue URL and the AWS credentials. You will need them later.

## Build
An Inlet is a connector which reads data from an external system and writes it to NATS. Let's start by creating a 
simple inlet connecting [AWS SQS](https://aws.amazon.com/sqs/) with NATS.

For this we will be using the `aws_sqs` source. You can find more information about the source by running the following 
command:
```shell
connect component describe source aws_sqs
```

### Generating an inlet
Nobody likes to write boilerplate code, so we have a generator to help you get started. Run the following command to
generate a new inlet and edit it:

```shell
connect connector create -i sqs-input
```
The `-i` tells the create command we want to interactively create the inlet. 

The CLI will ask if you want to create an inlet or an outlet. Select `inlet` and press enter. The CLI will generate a
template inlet definition and open it in your default editor.

Since we are creating an inlet, the steps in our config contains a `source` and `producer` section. The `source` 
section is where we define how to read data from the external system. The `producer` section is where we define how to 
write data to NATS.

Let's fill in the `source` and `producer` sections of our connector. Here is an example:

```yaml
description: A simple input reading from SQS
workload: ghcr.io/synadia-io/connect-runtime-wombat:latest
metrics:
    port: 4195
    path: /metrics
steps:
    source:
        type: aws_sqs
        config:
          url: https://sqs.us-east-2.amazonaws.com/563342913055/connect-test-1
          region: us-east-2
          credentials:
            id: YOUR_AWS_ACCESS_KEY_ID
            secret: YOUR_AWS_SECRET_ACCESS_KEY
    producer:
        subject: connect.inlet.aws.sqs
        nats_config:
            url: nats://demo.nats.io:4222
```

Obviously, replace the `url`, `region`, `id`, and `secret` fields with your own values. The `subject` field is the
NATS subject on which the messages will be published. The `nats_config.url` field is the URL of the NATS server.

Saving the file will automatically upload the inlet definition to the cloud. You can now continue deploying the inlet.
You can validate the inlet by running `connect connector get <connector-name>` or list the available connectors using
`connect connector list`.

## Deploy
Deploying the inlet is as simple as running `connect connector deploy <connector-name>`. In our case, we would run:

```shell
connect connector deploy sqs-input
```

You can check the status of the deployment by running `connect connector get <connector-name>`.