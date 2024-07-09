// Code generated by paramgen. DO NOT EDIT.
// Source: github.com/ConduitIO/conduit-commons/tree/main/paramgen

package log

import (
	"github.com/conduitio/conduit-commons/config"
)

func (DestinationConfig) Parameters() map[string]config.Parameter {
	return map[string]config.Parameter{
		"level": {
			Default:     "info",
			Description: "The log level used to log records.",
			Type:        config.ParameterTypeString,
			Validations: []config.Validation{
				config.ValidationInclusion{List: []string{"trace", "debug", "info", "warn", "error"}},
			},
		},
		"message": {
			Default:     "",
			Description: "Optional message that should be added to the log output of every record.",
			Type:        config.ParameterTypeString,
			Validations: []config.Validation{},
		},
	}
}