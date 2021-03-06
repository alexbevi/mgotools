package command

import (
	"bytes"
	"fmt"
	"time"

	"mgotools/command/format"
	"mgotools/parser"
	"mgotools/parser/context"
	"mgotools/record"
	"mgotools/util"
)

type restart struct {
	instance map[int]*restartInstance
}

type restartInstance struct {
	summary  format.LogSummary
	restarts []struct {
		Date    time.Time
		Startup record.MsgVersion
	}
}

func init() {
	GetFactory().Register("restart", Definition{}, func() (Command, error) {
		return &restart{make(map[int]*restartInstance)}, nil
	})
}

func (r *restart) Finish(index int, out commandTarget) error {
	instance := r.instance[index]
	writer := bytes.NewBuffer([]byte{})

	instance.summary.Print(writer)
	writer.WriteRune('\n')
	writer.WriteString("RESTARTS\n")

	for _, restart := range r.instance[index].restarts {
		writer.WriteString(fmt.Sprintf("   %s %s", restart.Date.Format(string(util.DATE_FORMAT_CTIMENOMS)), restart.Startup.String))
	}

	return nil
}

func (r *restart) Prepare(name string, index int, _ ArgumentCollection) error {
	r.instance[index] = &restartInstance{summary: format.NewLogSummary(name)}

	return nil
}

func (r *restart) Run(index int, out commandTarget, in commandSource, errors commandError) error {
	instance := r.instance[index]
	summary := &instance.summary

	// Create a local context object to create record.Entry objects.
	context := context.New(parser.VersionParserFactory.GetAll(), util.DefaultDateParser.Clone())
	defer context.Finish()

	for base := range in {
		entry, err := context.NewEntry(base)
		if err != nil {
			errors <- err
		}

		summary.Update(entry)

		if entry.Message == nil {
			continue
		} else if restart, ok := entry.Message.(record.MsgVersion); !ok {
			continue
		} else {
			instance.restarts = append(instance.restarts, struct {
				Date    time.Time
				Startup record.MsgVersion
			}{Date: entry.Date, Startup: restart})
		}
	}

	return nil
}

func (r *restart) Terminate(commandTarget) error {
	return nil
}
