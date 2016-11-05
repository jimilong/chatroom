package protocol

import (
	"bufio"
	"bytes"
	"encoding/binary"
	//"fmt"
	"hash/crc32"
)

type Header struct {
	BodyLen uint16 //2
	AskId   uint32 //4
	Service uint32 //4
}

func Pack(message []byte, service string, askId uint32) ([]byte, error) {
	head := new(Header)
	head.BodyLen = uint16(len(message))
	head.AskId = askId
	head.Service = crc32.ChecksumIEEE([]byte(service))

	var pkg *bytes.Buffer = new(bytes.Buffer)
	err := binary.Write(pkg, binary.BigEndian, head)
	if err != nil {
		return nil, err
	}

	err = binary.Write(pkg, binary.BigEndian, message)
	if err != nil {
		return nil, err
	}

	return pkg.Bytes(), nil
}

func Unpack(reader *bufio.Reader) ([]byte, *Header, error) {
	lengthByte, _ := reader.Peek(2) //读首两字节获取包长度
	lengthBuff := bytes.NewBuffer(lengthByte)

	var length uint16
	err := binary.Read(lengthBuff, binary.BigEndian, &length)
	if err != nil {
		return []byte(""), new(Header), err
	}

	if uint16(reader.Buffered()) < length+10 {
		return []byte(""), new(Header), err
	}

	pack := make([]byte, int(length+10))
	_, err = reader.Read(pack)
	if err != nil {
		return []byte(""), new(Header), err
	}

	head := new(Header)
	head.BodyLen = binary.BigEndian.Uint16(pack[:2])
	head.AskId = binary.BigEndian.Uint32(pack[2:6])
	head.Service = binary.BigEndian.Uint32(pack[6:10])
	//fmt.Println(head.BodyLen, "-", head.AskId, "-", head.Service)

	return pack[10:], head, nil
}
