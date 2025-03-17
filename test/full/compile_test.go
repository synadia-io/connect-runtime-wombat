package full_test

import (
    "fmt"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "github.com/synadia-io/connect-runtime-wombat/compiler"
    "github.com/synadia-io/connect-runtime-wombat/test"
    "github.com/synadia-io/connect/convert"
    "github.com/synadia-io/connect/spec"
    "gopkg.in/yaml.v3"
)

var _ = Describe("Compile", func() {
    When("the connector configuration is valid", func() {
        Context("for Generate inlet", func() {
            It("should compile without error", func() {
                in := `
description: A summary of what this connector does
runtime_id: wombat
steps:
    producer:
        core: 
            subject: connect.demo
        nats:
            url: nats://demo.nats.io:4222
        threads: 1
    source:
        config:
            interval: 1s
            mapping: root.message = "Hello, World!"
        type: generate
`
                exp := `
input:
    generate:
        interval: 1s
        mapping: root.message = "Hello, World!"
output:
    nats:
        urls:
            - nats://demo.nats.io:4222
        subject: connect.demo
        max_in_flight: 1
        metadata:
            include_patterns: [".*"]
`
                testConfig(in, exp)
            })
        })
    })
})

func testConfig(cfg string, exp string) {
    var sp spec.ConnectorSpec
    if err := yaml.Unmarshal([]byte(cfg), &sp); err != nil {
        Fail(fmt.Sprintf("could not parse yaml: %v", err))
    }

    msp := convert.ConvertStepsFromSpec(sp.Steps)

    res, err := compiler.Compile(test.Runtime(), msp)
    Expect(err).To(BeNil())

    var rm map[string]any
    if err := yaml.Unmarshal([]byte(res), &rm); err != nil {
        Fail(fmt.Sprintf("could not parse yaml: %v", err))
    }

    var me map[string]any
    if err := yaml.Unmarshal([]byte(exp), &me); err != nil {
        Fail(fmt.Sprintf("could not parse yaml: %v", err))
    }

    Expect(rm["input"]).To(Equal(me["input"]))
    Expect(rm["output"]).To(Equal(me["output"]))
}
