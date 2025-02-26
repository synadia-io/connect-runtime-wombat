package compiler

import (
    "github.com/synadia-io/connect/model"
)

func compileTransformer(transformer model.TransformerStep) Fragment {
    if transformer.Composite != nil {
        return compileCompositeTransformer(transformer.Composite)
    }

    if transformer.Service != nil {
        return compileServiceTransformer(transformer.Service)
    }

    if transformer.Mapping != nil {
        return compileMappingTransformer(transformer.Mapping)
    }

    return nil
}

func compileServiceTransformer(t *model.ServiceTransformerStep) Fragment {
    return Frag().
        Fragment("nats_request_reply", natsBaseFragment(t.Nats).
            String("subject", t.Endpoint).
            String("timeout", t.Timeout).
            Fragment("metadata", Frag().
                Strings("include_patterns", ".*")))
}

func compileCompositeTransformer(t *model.CompositeTransformerStep) Fragment {
    var seq []Fragment
    for _, ct := range t.Sequential {
        seq = append(seq, compileTransformer(ct))
    }

    return Frag().Fragment("processors", Frag().Fragments("sequence", seq...))
}

func compileMappingTransformer(t *model.MappingTransformerStep) Fragment {
    return Frag().String("mapping", t.Sourcecode)
}
