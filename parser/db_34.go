package parser

import "mgotools/util"

type LogVersion34Parser struct {
	LogVersionCommon
}

func (v *LogVersion34Parser) NewLogMessage(entry LogEntry) (LogMessage, error) {
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
	return nil, LogVersionErrorUnmatched{Message: "version 3.4"}
}
func (v *LogVersion34Parser) Version() LogVersionDefinition {
	return LogVersionDefinition{Major: 3, Minor: 4, Binary: LOG_VERSION_MONGOD}
}