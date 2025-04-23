package main

import (
    "github.com/redpanda-data/benthos/v4/public/service"
    "github.com/stretchr/testify/assert"
    "testing"
)

func TestFieldTreeConstruction(t *testing.T) {
    templateFields := []service.TemplateDataPluginField{
        {FullName: "addresses", IsSecret: true},
        {FullName: "tls", IsSecret: true},
        {FullName: "tls.enabled", IsSecret: true},
        {FullName: "tls.certs[].cert.pem", IsSecret: true},
        {FullName: "tls.certs", IsSecret: true},
        {FullName: "tls.certs[].cert", IsSecret: true},
    }

    cc := fieldTree(templateFields)
    assert.Equal(t, 2, len(cc.children))

    addresses := cc.Get([]string{"addresses"})
    assert.Equal(t, addresses.Field.FullName, "addresses")
    assert.Truef(t, addresses.Field.IsSecret, "missing container state")

    tls := cc.Get([]string{"tls"})
    assert.NotNil(t, tls)
    assert.Equal(t, tls.Field.FullName, "tls")
    assert.Truef(t, tls.Field.IsSecret, "missing container state")
    assert.Equal(t, 2, len(tls.children))

    tlsEnabled := cc.Get([]string{"tls", "enabled"})
    assert.Equal(t, tlsEnabled.Field.FullName, "tls.enabled")
    assert.Truef(t, tlsEnabled.Field.IsSecret, "missing container state")

    tlsCerts := cc.Get([]string{"tls", "certs"})
    assert.Equal(t, tlsCerts.Field.FullName, "tls.certs")
    assert.Truef(t, tlsCerts.Field.IsSecret, "missing container state")
    assert.Equal(t, 1, len(tlsCerts.children))
}
