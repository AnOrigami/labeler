package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/klauspost/compress/gzip"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

type GzipJSON []byte

func (gj GzipJSON) MarshalJSON() ([]byte, error) {
	if gj == nil {
		return []byte("null"), nil
	}
	return gj, nil
}

func (gj *GzipJSON) UnmarshalJSON(data []byte) error {
	*gj = data
	return nil
}

type JSONCodec struct{}

var (
	GzipJSONType = reflect.TypeOf(GzipJSON{})
)

func (jc *JSONCodec) DecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != GzipJSONType {
		return bsoncodec.ValueDecoderError{Name: "GzipJSONDecodeValue", Types: []reflect.Type{GzipJSONType}, Received: val}
	}
	vrType := vr.Type()
	switch vrType {
	case bsontype.EmbeddedDocument:
		var m bson.M
		dc, err := bson.NewDecoder(vr)
		if err != nil {
			return err
		}
		if err := dc.Decode(&m); err != nil {
			return err
		}
		b, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return err
		}
		val.Set(reflect.ValueOf(GzipJSON(b)))
	case bsontype.Binary:
		b, _, err := vr.ReadBinary()
		if err != nil {
			return err
		}
		gzipReader, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			return err
		}
		defer gzipReader.Close()
		decompressed, err := io.ReadAll(gzipReader)
		if err != nil {
			return err
		}
		val.Set(reflect.ValueOf(decompressed))
	default:
		return fmt.Errorf("connot decode into GzipJSON")
	}
	return nil
}

func (jc *JSONCodec) EncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != GzipJSONType {
		return bsoncodec.ValueEncoderError{Name: "GzipJSONEncodeValue", Types: []reflect.Type{GzipJSONType}, Received: val}
	}
	compressed := new(bytes.Buffer)
	gzipWriter := gzip.NewWriter(compressed)
	_, err := gzipWriter.Write(val.Interface().(GzipJSON))
	if err != nil {
		return err
	}
	if err := gzipWriter.Close(); err != nil {
		return err
	}
	return vw.WriteBinary(compressed.Bytes())
}
