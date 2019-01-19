package logger

import (
	"net"
	"strconv"
	"strings"

	"mgotools/internal"
	"mgotools/mongo"
	"mgotools/record"
	"mgotools/util"
)

func Control(r util.RuneReader, entry record.Entry) (record.Message, error) {
	switch entry.Context {
	case "initandlisten":
		switch {
		case r.ExpectString("build info"):
			return record.MsgBuildInfo{BuildInfo: r.SkipWords(2).Remainder()}, nil
		case r.ExpectString("db version"):
			return Version(r.SkipWords(2).Remainder(), "mongod")
		case r.ExpectString("MongoDB starting"):
			return StartupInfo(entry.RawMessage)
		case r.ExpectString("OpenSSL version"):
			return Version(r.SkipWords(2).Remainder(), "OpenSSL")
		case r.ExpectString("options"):
			r.SkipWords(1)
			return StartupOptions(r.Remainder())
		}

	case "signalProcessingThread":
		if r.ExpectString("dbexit") {
			return record.MsgShutdown{String: r.Remainder()}, nil
		} else {
			return record.MsgSignal{String: r.Remainder()}, nil
		}
	}
	return nil, internal.ControlUnrecognized
}

func Network(r util.RuneReader, entry record.Entry) (record.Message, error) {
	if entry.Connection == 0 {
		if r.ExpectString("connection accepted") { // connection accepted from <IP>:<PORT> #<CONN>
			if addr, port, conn, ok := connectionInit(r.SkipWords(2)); ok {
				return record.MsgConnection{Address: addr, Port: port, Conn: conn, Opened: true}, nil
			}
		} else if r.ExpectString("waiting for connections") {
			return record.MsgListening{}, nil
		} else if entry.Context == "signalProcessingThread" {
			return record.MsgSignal{String: entry.RawMessage}, nil
		}
	} else if entry.Connection > 0 {
		if r.ExpectString("end connection") {
			if addr, port, _, ok := connectionInit(&r); ok {
				return record.MsgConnection{Address: addr, Port: port, Conn: entry.Connection, Opened: false}, nil
			}
		} else if r.ExpectString("received client metadata from") {
			// Skip "received client metadata" and grab connection information.
			if addr, port, conn, ok := connectionInit(r.SkipWords(3)); !ok {
				return nil, internal.MetadataUnmatched
			} else {
				if meta, err := mongo.ParseJsonRunes(&r, false); err == nil {
					return record.MsgConnectionMeta{
						MsgConnection: record.MsgConnection{
							Address: addr,
							Conn:    conn,
							Port:    port,
							Opened:  true},
						Meta: meta}, nil
				}
			}
		}
	}

	return nil, internal.NetworkUnrecognized
}

func Storage(r util.RuneReader, entry record.Entry) (record.Message, error) {
	switch entry.Context {
	case "signalProcessingThread":
		return record.MsgSignal{entry.RawMessage}, nil

	case "initandlisten":
		if r.ExpectString("wiredtiger_open config") {
			return record.MsgWiredTigerConfig{String: r.SkipWords(2).Remainder()}, nil
		}
	}

	return nil, internal.StorageUnmatched
}

func connectionInit(msg *util.RuneReader) (net.IP, uint16, int, bool) {
	var (
		addr   *util.RuneReader
		buffer string
		char   rune
		conn          = 0
		port          = 0
		ip     net.IP = nil
		ok     bool
	)

	msg.SkipWords(1) // "from"
	partialAddress, ok := msg.SlurpWord()
	if !ok {
		return nil, 0, 0, false
	}

	addr = util.NewRuneReader(partialAddress)
	length := addr.Length()
	addr.Seek(length, 0)
	for {
		if char, ok = addr.Prev(); !ok || char == ':' {
			addr.Next()
			break
		}
	}

	pos := addr.Pos()
	if buffer, ok = addr.Substr(pos, length-pos); ok {
		port, _ = strconv.Atoi(buffer)
	}

	if part, ok := addr.Substr(0, pos-1); ok {
		ip = net.ParseIP(part)
	}

	if msgNumber, ok := msg.SlurpWord(); ok {
		if msgNumber[0] == '#' {
			conn, _ = strconv.Atoi(msgNumber[1:])
		} else if len(msgNumber) > 4 && strings.HasPrefix(msgNumber, "conn") {
			if strings.HasSuffix(msgNumber, ":") {
				msgNumber = msgNumber[:len(msgNumber)-1]
			}
			conn, _ = strconv.Atoi(msgNumber[4:])
		}
	}

	return ip, uint16(port), conn, true
}

func StartupInfo(msg string) (record.MsgStartupInfo, error) {
	if optionsRegex, err := util.GetRegexRegistry().Compile(`([^=\s]+)=([^\s]+)`); err == nil {
		matches := optionsRegex.FindAllStringSubmatch(msg, -1)
		startupInfo := record.MsgStartupInfo{}

		for _, match := range matches {
			switch match[1] {
			case "dbpath":
				startupInfo.DbPath = match[2]
			case "host":
				startupInfo.Hostname = match[2]
			case "pid":
				startupInfo.Pid, _ = strconv.Atoi(match[2])
			case "port":
				startupInfo.Port, _ = strconv.Atoi(match[2])
			}
		}
		return startupInfo, nil
	}
	return record.MsgStartupInfo{}, internal.NoStartupArgumentsFound
}

func StartupOptions(msg string) (record.MsgStartupOptions, error) {
	opt, err := mongo.ParseJson(msg, false)
	if err != nil {
		return record.MsgStartupOptions{}, err
	}
	return record.MsgStartupOptions{String: msg, Options: opt}, nil
}

func Version(msg string, binary string) (record.MsgVersion, error) {
	msg = strings.TrimLeft(msg, "v")
	version := record.MsgVersion{String: msg, Binary: binary}
	if parts := strings.Split(version.String, "."); len(parts) >= 2 {
		version.Major, _ = strconv.Atoi(parts[0])
		version.Minor, _ = strconv.Atoi(parts[1])
		if len(parts) >= 3 {
			version.Revision, _ = strconv.Atoi(parts[2])
		}
	}
	if version.String == "" {
		return version, internal.UnexpectedVersionFormat
	}
	return version, nil
}
