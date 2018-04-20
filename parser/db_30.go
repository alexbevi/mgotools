package parser

import (
	"mgotools/log"
	"mgotools/util"
)

type LogVersion30Parser struct {
	LogVersionCommon
}

func init() {
	LogVersionParserFactory.Register(func() LogVersionParser {
		return &LogVersion30Parser{LogVersionCommon{util.NewDateParser([]string{util.DATE_FORMAT_ISO8602_UTC, util.DATE_FORMAT_ISO8602_LOCAL})}}
	})
}

func (v *LogVersion30Parser) NewLogMessage(entry log.Entry) (log.Message, error) {
	r := *util.NewRuneReader(entry.RawMessage)
	switch entry.RawComponent {
	case "COMMAND", "WRITE":
		if msg, err := parse3XCommand(r, true); err == nil {
			return msg, err
		} else if msg, err := v.ParseDDL(r, entry); err == nil {
			return msg, nil
		}
	case "INDEX":
		if r.ExpectString("build index on") {
			return parse3XBuildIndex(r)
		}
	case "CONTROL":
		return v.ParseControl(r, entry)
	case "NETWORK":
		return v.ParseNetwork(r, entry)
	case "STORAGE":
		return v.ParseStorage(r, entry)
	}
	return nil, LogVersionErrorUnmatched{"version 3.0"}
}
func (v *LogVersion30Parser) Version() LogVersionDefinition {
	return LogVersionDefinition{Major: 3, Minor: 0, Binary: LOG_VERSION_MONGOD}
}
