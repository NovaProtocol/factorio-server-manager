package factorio

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

// ModSettings is a parsed mod-settings.dat: the version that wrote it and the
// property tree holding the three sections.
type ModSettings struct {
	Version  Version
	Sections SettingsDict
}

// SettingsDict keeps a dictionary's entries in the order the file had them. A Go
// map cannot promise that order, and the encoder has to reproduce it to
// round-trip the file byte for byte.
type SettingsDict []SettingsEntry

type SettingsEntry struct {
	Key   string
	Value interface{}
}

// MarshalJSON writes the entries as a JSON object, in their stored order.
func (d SettingsDict) MarshalJSON() ([]byte, error) {
	var out bytes.Buffer

	out.WriteByte('{')
	for i, entry := range d {
		if i > 0 {
			out.WriteByte(',')
		}

		key, err := json.Marshal(entry.Key)
		if err != nil {
			return nil, err
		}
		out.Write(key)
		out.WriteByte(':')

		value, err := json.Marshal(entry.Value)
		if err != nil {
			return nil, err
		}
		out.Write(value)
	}
	out.WriteByte('}')

	return out.Bytes(), nil
}

func (d SettingsDict) Get(key string) (interface{}, bool) {
	for _, entry := range d {
		if entry.Key == key {
			return entry.Value, true
		}
	}
	return nil, false
}

func (d *SettingsDict) Set(key string, value interface{}) {
	for i := range *d {
		if (*d)[i].Key == key {
			(*d)[i].Value = value
			return
		}
	}
	*d = append(*d, SettingsEntry{Key: key, Value: value})
}

// Factorio keeps its own settings in mod-settings.dat beside the data directory:
// the startup, runtime-global (map) and runtime-per-user settings of every mod
// that declares any. The layout is the one documented on the wiki (Mod settings
// file format): an 8 byte version, then from 0.17 one unused byte, then a single
// property tree whose keys are the three sections.
//
// The file only exists once the game has written it, so absent is a normal
// state rather than an error, and it is never generated here.

const (
	treeNone   byte = 0
	treeBool   byte = 1
	treeDouble byte = 2
	treeString byte = 3
	treeList   byte = 4
	treeDict   byte = 5
	// treeInt is a 64 bit integer. 2.1 added it for settings values, and neither
	// of the parsers this was ported from knows about it, so a 2.1 file stops
	// dead on the first one without it.
	treeInt byte = 6
)

type settingsTree struct {
	reader  io.Reader
	version Version
}

func (s *settingsTree) u8() (byte, error) {
	var b [1]byte
	if _, err := io.ReadFull(s.reader, b[:]); err != nil {
		return 0, err
	}
	return b[0], nil
}

func (s *settingsTree) u16() (uint16, error) {
	var b [2]byte
	if _, err := io.ReadFull(s.reader, b[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b[:]), nil
}

func (s *settingsTree) u32() (uint32, error) {
	var b [4]byte
	if _, err := io.ReadFull(s.reader, b[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b[:]), nil
}

func (s *settingsTree) double() (float64, error) {
	var b [8]byte
	if _, err := io.ReadFull(s.reader, b[:]); err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(b[:])), nil
}

func (s *settingsTree) i64() (int64, error) {
	var b [8]byte
	if _, err := io.ReadFull(s.reader, b[:]); err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(b[:])), nil
}

// A "optim" integer is one byte holding the value, unless that byte is 0xFF in
// which case the full four bytes follow.
func (s *settingsTree) optim() (uint32, error) {
	b, err := s.u8()
	if err != nil {
		return 0, err
	}
	if b != 0xFF {
		return uint32(b), nil
	}
	return s.u32()
}

// Strings in this file carry a leading "is empty" byte, which the strings in a
// save header do not.
func (s *settingsTree) str() (string, error) {
	empty, err := s.u8()
	if err != nil {
		return "", err
	}
	if empty != 0 {
		return "", nil
	}

	length, err := s.optim()
	if err != nil {
		return "", err
	}
	if length == 0 {
		return "", nil
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(s.reader, buf); err != nil {
		return "", err
	}

	return string(buf), nil
}

func (s *settingsTree) tree() (interface{}, error) {
	kind, err := s.u8()
	if err != nil {
		return nil, err
	}

	// One byte Factorio describes as not important outside its internals.
	if _, err := s.u8(); err != nil {
		return nil, err
	}

	switch kind {
	case treeNone:
		return nil, nil

	case treeBool:
		b, err := s.u8()
		if err != nil {
			return nil, err
		}
		return b != 0, nil

	case treeDouble:
		return s.double()

	case treeInt:
		return s.i64()

	case treeString:
		return s.str()

	case treeList:
		length, err := s.u32()
		if err != nil {
			return nil, err
		}

		list := make([]interface{}, 0, length)
		for i := uint32(0); i < length; i++ {
			if _, err := s.str(); err != nil { // the entry name, unused
				return nil, err
			}

			item, err := s.tree()
			if err != nil {
				return nil, err
			}
			list = append(list, item)
		}
		return list, nil

	case treeDict:
		length, err := s.u32()
		if err != nil {
			return nil, err
		}

		dict := make(SettingsDict, 0, length)
		for i := uint32(0); i < length; i++ {
			key, err := s.str()
			if err != nil {
				return nil, err
			}

			value, err := s.tree()
			if err != nil {
				return nil, err
			}
			dict = append(dict, SettingsEntry{Key: key, Value: value})
		}
		return dict, nil
	}

	return nil, fmt.Errorf("unknown property tree type %d", kind)
}

// ModSettingsPath is where this install keeps mod-settings.dat. The game writes
// it inside the mods directory, beside the mods the settings belong to; an older
// layout kept it at the root of the data directory, so that is checked second.
func ModSettingsPath() string {
	config := bootstrap.GetConfig()

	inMods := filepath.Join(config.FactorioModsDir, "mod-settings.dat")
	if _, err := os.Stat(inMods); err == nil {
		return inMods
	}

	return filepath.Join(config.FactorioDir, "mod-settings.dat")
}

// LoadModSettingsFile reads mod-settings.dat. A missing file returns nil and no
// error: the game simply has not written one yet.
func LoadModSettingsFile(path string) (*ModSettings, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := &settingsTree{reader: file}

	var raw [8]byte
	if _, err := io.ReadFull(file, raw[:]); err != nil {
		return nil, fmt.Errorf("read the mod settings version: %v", err)
	}
	reader.version = Version{
		uint(binary.LittleEndian.Uint16(raw[0:2])),
		uint(binary.LittleEndian.Uint16(raw[2:4])),
		uint(binary.LittleEndian.Uint16(raw[4:6])),
		uint(binary.LittleEndian.Uint16(raw[6:8])),
	}

	if reader.version.Greater(Version{0, 17, 0, 0}) {
		if _, err := reader.u8(); err != nil {
			return nil, fmt.Errorf("read the 0.17 marker byte: %v", err)
		}
	}

	root, err := reader.tree()
	if err != nil {
		return nil, err
	}

	sections, ok := root.(SettingsDict)
	if !ok {
		return nil, fmt.Errorf("the mod settings root is not a dictionary")
	}

	return &ModSettings{Version: reader.version, Sections: sections}, nil
}

// EncodeModSettings writes settings back in the game's own format. The reader is
// the specification here: encoding what was just parsed has to reproduce the
// original file byte for byte, which is what the round-trip test checks before
// anything is allowed to write.
func EncodeModSettings(settings *ModSettings) ([]byte, error) {
	var out bytes.Buffer

	var header [8]byte
	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint16(header[i*2:i*2+2], uint16(settings.Version[i]))
	}
	out.Write(header[:])

	if settings.Version.Greater(Version{0, 17, 0, 0}) {
		out.WriteByte(0)
	}

	if err := writeTree(&out, settings.Sections); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func writeTree(out *bytes.Buffer, value interface{}) error {
	switch typed := value.(type) {
	case nil:
		out.WriteByte(treeNone)
		out.WriteByte(0)

	case bool:
		out.WriteByte(treeBool)
		out.WriteByte(0)
		if typed {
			out.WriteByte(1)
		} else {
			out.WriteByte(0)
		}

	case float64:
		out.WriteByte(treeDouble)
		out.WriteByte(0)
		var raw [8]byte
		binary.LittleEndian.PutUint64(raw[:], math.Float64bits(typed))
		out.Write(raw[:])

	case int64:
		out.WriteByte(treeInt)
		out.WriteByte(0)
		var raw [8]byte
		binary.LittleEndian.PutUint64(raw[:], uint64(typed))
		out.Write(raw[:])

	case int:
		return writeTree(out, int64(typed))

	case string:
		out.WriteByte(treeString)
		out.WriteByte(0)
		writeString(out, typed)

	case []interface{}:
		out.WriteByte(treeList)
		out.WriteByte(0)
		var raw [4]byte
		binary.LittleEndian.PutUint32(raw[:], uint32(len(typed)))
		out.Write(raw[:])
		for _, item := range typed {
			writeString(out, "")
			if err := writeTree(out, item); err != nil {
				return err
			}
		}

	case SettingsDict:
		out.WriteByte(treeDict)
		out.WriteByte(0)
		var raw [4]byte
		binary.LittleEndian.PutUint32(raw[:], uint32(len(typed)))
		out.Write(raw[:])
		for _, entry := range typed {
			writeString(out, entry.Key)
			if err := writeTree(out, entry.Value); err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("cannot encode a %T in mod settings", value)
	}

	return nil
}

// The leading byte is kept at zero even for an empty string: the game does not
// use it, and writing the two byte form is what reproduces the original file.
func writeString(out *bytes.Buffer, value string) {
	out.WriteByte(0)
	writeOptim(out, uint32(len(value)))
	out.WriteString(value)
}

// writeOptim mirrors the reader: one byte unless the value does not fit in one,
// in which case 0xFF introduces the full four.
func writeOptim(out *bytes.Buffer, value uint32) {
	if value < 0xFF {
		out.WriteByte(byte(value))
		return
	}

	out.WriteByte(0xFF)
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], value)
	out.Write(raw[:])
}
