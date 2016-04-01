package utils

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"hash/fnv"
	"strconv"
)

//string to hash
func MakeHash(s string) string {
	const IEEE = 0xedb88320
	var IEEETable = crc32.MakeTable(IEEE)
	hash := fmt.Sprintf("%x", crc32.Checksum([]byte(s), IEEETable))
	return hash
}

func HashString(encode string) uint64 {
	hash := fnv.New64()
	hash.Write([]byte(encode))
	return hash.Sum64()
}

// 制作特征值方法一
func MakeUnique(obj interface{}) string {
	baseString, _ := json.Marshal(obj)
	return strconv.FormatUint(HashString(string(baseString)), 10)
}

// 制作特征值方法二
func MakeMd5(obj interface{}, length ...int) string {
	h := md5.New()
	baseString, _ := json.Marshal(obj)
	h.Write([]byte(baseString))
	s := hex.EncodeToString(h.Sum(nil))
	if len(length) == 0 {
		return s
	}
	if length[0] > 32 {
		length[0] = 32
	}
	return s[:length[0]]
}
