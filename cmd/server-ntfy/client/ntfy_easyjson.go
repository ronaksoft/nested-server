// Code generated by easyjson for marshaling/unmarshaling. DO NOT EDIT.

package ntfy

import (
	json "encoding/json"
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient(in *jlexer.Lexer, out *CMDUnRegisterWebsocket) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "ws_id":
			out.WebsocketID = string(in.String())
		case "bundle_id":
			out.BundleID = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient(out *jwriter.Writer, in CMDUnRegisterWebsocket) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"ws_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.WebsocketID))
	}
	{
		const prefix string = ",\"bundle_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.BundleID))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v CMDUnRegisterWebsocket) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v CMDUnRegisterWebsocket) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *CMDUnRegisterWebsocket) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *CMDUnRegisterWebsocket) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient(l, v)
}
func easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient1(in *jlexer.Lexer, out *CMDUnRegisterDevice) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "_did":
			out.DeviceID = string(in.String())
		case "_dt":
			out.DeviceToken = string(in.String())
		case "uid":
			out.UserID = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient1(out *jwriter.Writer, in CMDUnRegisterDevice) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"_did\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.DeviceID))
	}
	{
		const prefix string = ",\"_dt\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.DeviceToken))
	}
	{
		const prefix string = ",\"uid\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.UserID))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v CMDUnRegisterDevice) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient1(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v CMDUnRegisterDevice) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient1(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *CMDUnRegisterDevice) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient1(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *CMDUnRegisterDevice) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient1(l, v)
}
func easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient2(in *jlexer.Lexer, out *CMDRegisterWebsocket) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "uid":
			out.UserID = string(in.String())
		case "ws_id":
			out.WebsocketID = string(in.String())
		case "bundle_id":
			out.BundleID = string(in.String())
		case "_did":
			out.DeviceID = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient2(out *jwriter.Writer, in CMDRegisterWebsocket) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"uid\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.UserID))
	}
	{
		const prefix string = ",\"ws_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.WebsocketID))
	}
	{
		const prefix string = ",\"bundle_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.BundleID))
	}
	{
		const prefix string = ",\"_did\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.DeviceID))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v CMDRegisterWebsocket) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient2(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v CMDRegisterWebsocket) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient2(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *CMDRegisterWebsocket) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient2(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *CMDRegisterWebsocket) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient2(l, v)
}
func easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient3(in *jlexer.Lexer, out *CMDRegisterDevice) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "_did":
			out.DeviceID = string(in.String())
		case "uid":
			out.UserID = string(in.String())
		case "_dt":
			out.DeviceToken = string(in.String())
		case "_os":
			out.DeviceOS = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient3(out *jwriter.Writer, in CMDRegisterDevice) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"_did\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.DeviceID))
	}
	{
		const prefix string = ",\"uid\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.UserID))
	}
	{
		const prefix string = ",\"_dt\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.DeviceToken))
	}
	{
		const prefix string = ",\"_os\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.DeviceOS))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v CMDRegisterDevice) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient3(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v CMDRegisterDevice) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient3(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *CMDRegisterDevice) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient3(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *CMDRegisterDevice) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient3(l, v)
}
func easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient4(in *jlexer.Lexer, out *CMDPushInternal) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "targets":
			if in.IsNull() {
				in.Skip()
				out.Targets = nil
			} else {
				in.Delim('[')
				if out.Targets == nil {
					if !in.IsDelim(']') {
						out.Targets = make([]string, 0, 4)
					} else {
						out.Targets = []string{}
					}
				} else {
					out.Targets = (out.Targets)[:0]
				}
				for !in.IsDelim(']') {
					var v1 string
					v1 = string(in.String())
					out.Targets = append(out.Targets, v1)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "local_only":
			out.LocalOnly = bool(in.Bool())
		case "msg":
			out.Message = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient4(out *jwriter.Writer, in CMDPushInternal) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"targets\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		if in.Targets == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v2, v3 := range in.Targets {
				if v2 > 0 {
					out.RawByte(',')
				}
				out.String(string(v3))
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"local_only\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Bool(bool(in.LocalOnly))
	}
	{
		const prefix string = ",\"msg\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Message))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v CMDPushInternal) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient4(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v CMDPushInternal) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient4(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *CMDPushInternal) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient4(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *CMDPushInternal) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient4(l, v)
}
func easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient5(in *jlexer.Lexer, out *CMDPushExternal) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "targets":
			if in.IsNull() {
				in.Skip()
				out.Targets = nil
			} else {
				in.Delim('[')
				if out.Targets == nil {
					if !in.IsDelim(']') {
						out.Targets = make([]string, 0, 4)
					} else {
						out.Targets = []string{}
					}
				} else {
					out.Targets = (out.Targets)[:0]
				}
				for !in.IsDelim(']') {
					var v4 string
					v4 = string(in.String())
					out.Targets = append(out.Targets, v4)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "data":
			if in.IsNull() {
				in.Skip()
			} else {
				in.Delim('{')
				if !in.IsDelim('}') {
					out.Data = make(map[string]string)
				} else {
					out.Data = nil
				}
				for !in.IsDelim('}') {
					key := string(in.String())
					in.WantColon()
					var v5 string
					v5 = string(in.String())
					(out.Data)[key] = v5
					in.WantComma()
				}
				in.Delim('}')
			}
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient5(out *jwriter.Writer, in CMDPushExternal) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"targets\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		if in.Targets == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v6, v7 := range in.Targets {
				if v6 > 0 {
					out.RawByte(',')
				}
				out.String(string(v7))
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"data\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		if in.Data == nil && (out.Flags&jwriter.NilMapAsEmpty) == 0 {
			out.RawString(`null`)
		} else {
			out.RawByte('{')
			v8First := true
			for v8Name, v8Value := range in.Data {
				if v8First {
					v8First = false
				} else {
					out.RawByte(',')
				}
				out.String(string(v8Name))
				out.RawByte(':')
				out.String(string(v8Value))
			}
			out.RawByte('}')
		}
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v CMDPushExternal) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient5(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v CMDPushExternal) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonD9ef3c31EncodeGitRonaksoftwareComNestedServerNtfyClient5(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *CMDPushExternal) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient5(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *CMDPushExternal) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonD9ef3c31DecodeGitRonaksoftwareComNestedServerNtfyClient5(l, v)
}