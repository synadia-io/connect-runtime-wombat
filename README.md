# Synadia Connect - Wombat Runtime
A runtime for Synadia Connect that provides [Wombat](https://wombat.dev) components for use in connectors.

Why would we want to reinvent the wheel while there is already a great project providing us with a lot of components?
Since Synadia Connect provides an abstraction called `Runtimes`. Runtimes provide the components you can use as part of
your connectors. Not only that, but a runtime is also responsible for running the connector.

You can leverage this runtime abstraction to build your own runtime, or like this project does, wrap an existing project
to benefit from the components it already provides.

## Synadia Connect
Synadia Connect is a platform for building and deploying connectors that move data between Nats and external systems. 
Connectors are built using a simple YAML configuration file and can be deployed to the Synadia Connect platform.

## Wombat
Wombat is a partial fork of the project previously known as Benthos. It relies on RedPanda Benthos for the core running
the connectors and incorporates all components from RedPanda Connect and adds forked components which RedPanda changed
the license on.

It also contains a bunch of new components that are not in Benthos or RedPanda Connect.

For more information on Wombat, please visit the [Wombat website](https://wombat.dev).